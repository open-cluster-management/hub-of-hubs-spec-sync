// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package sync

import (
	"context"
	"time"

	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pgx "github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/open-cluster-management/hub-of-hubs-spec-syncer/pkg/controller/sync/utils"
)

const (
	controllerName = "policy-spec-syncer"
	finalizerName  = "hub-of-hubs.open-cluster-management.io/policy-cleanup"
)

var log = ctrl.Log.WithName(controllerName)

// Add creates a new Policy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	return add(mgr, newReconciler(mgr, databaseConnectionPool))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) reconcile.Reconciler {
	return &ReconcilePolicy{client: mgr.GetClient(), scheme: mgr.GetScheme(), databaseConnectionPool: databaseConnectionPool}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr ctrl.Manager, r reconcile.Reconciler) error {
     return ctrl.NewControllerManagedBy(mgr).For(&policiesv1.Policy{}).Complete(r)
}

// blank assignment to verify that ReconcilePolicy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcilePolicy{}

// ReconcilePolicy reconciles a Policy object
type ReconcilePolicy struct {
	client                 client.Client
	scheme                 *runtime.Scheme
	databaseConnectionPool *pgxpool.Pool
}

// Reconcile reads that state of the cluster for a Policy object and makes changes based on the state read
// and what is in the Policy.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePolicy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Policy...")

	ctx := context.Background()

	// Fetch the Policy instance
	instance := &policiesv1.Policy{}
	err := r.client.Get(ctx, request.NamespacedName, instance)

	if err != nil {
		if errors.IsNotFound(err) {
			// the policy on hub was deleted, update all the matching policies in the database as deleted
			err = r.deleteFromTheDatabase(request.Name, request.Namespace)
			if err != nil {
				log.Error(err, "Delete failed")
				return reconcile.Result{}, err
			}

			reqLogger.Info("Reconciliation complete.")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get policy from hub...")
		return reconcile.Result{}, err
	}

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !utils.ContainsString(instance.GetFinalizers(), finalizerName) {
			reqLogger.Info("Adding finalizer")
			controllerutil.AddFinalizer(instance, finalizerName)
			if err := r.client.Update(ctx, instance); err != nil {
				return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, err
			}
		}
	} else {
		if utils.ContainsString(instance.GetFinalizers(), finalizerName) {
			// the policy is being deleted, update all the matching policies in the database as deleted
			if err := r.deleteFromTheDatabase(request.Name, request.Namespace); err != nil {
				log.Error(err, "Delete failed")
				return reconcile.Result{}, err
			}
			reqLogger.Info("Removing finalizer")
			controllerutil.RemoveFinalizer(instance, finalizerName)
			if err = r.client.Update(ctx, instance); err != nil {
				return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, err
			}
		}
		reqLogger.Info("Reconciliation complete.")
		return reconcile.Result{}, nil
	}

	// clean the instance
	instance.ResourceVersion = ""
	instance.ManagedFields = nil
	instance.Finalizers = nil

	instanceInTheDatabase := &policiesv1.Policy{}
	err = r.databaseConnectionPool.QueryRow(context.Background(),
		`SELECT payload FROM spec.policies WHERE id = $1`, string(instance.UID)).Scan(&instanceInTheDatabase)

	if err == pgx.ErrNoRows {
		reqLogger.Info("The Policy with the current UID does not exist in the database, inserting...")
		_, err := r.databaseConnectionPool.Exec(context.Background(),
			"INSERT INTO spec.policies (id,payload) values($1, $2::jsonb)", string(instance.UID), &instance)
		if err != nil {
			log.Error(err, "Insert failed")
		} else {
			reqLogger.Info("Policy has been inserted into the database...Reconciliation complete.")
		}
		return reconcile.Result{}, err
	}

	// found, then compare and update
	if !utils.CompareInstances(instance, instanceInTheDatabase) {
		reqLogger.Info("Policy mismatch between hub and the database, updating the database...")

		if _, err := r.databaseConnectionPool.Exec(context.Background(),
			`UPDATE spec.policies SET payload = $1 WHERE id = $2`, &instance, string(instance.UID)); err != nil {
			log.Error(err, "Update failed")
			return reconcile.Result{}, err
		}
	}

	reqLogger.Info("Reconciliation complete.")
	return reconcile.Result{}, err
}

func (r *ReconcilePolicy) deleteFromTheDatabase(name, namespace string) error {
	reqLogger := log.WithValues("Request.Namespace", namespace, "Request.Name", name)
	// the policy on hub was deleted, update all the matching policies in the database as deleted
	reqLogger.Info("Policy was deleted, update the deleted field in the database...")

	_, err := r.databaseConnectionPool.Exec(context.Background(),
		`UPDATE spec.policies SET deleted = true WHERE payload -> 'metadata' ->> 'name' = $1 AND
			     payload -> 'metadata' ->> 'namespace' = $2 AND deleted = false`, name, namespace)

	if err == nil {
		reqLogger.Info("Policy has been updated as deleted in the database...")
	}
	return err
}
