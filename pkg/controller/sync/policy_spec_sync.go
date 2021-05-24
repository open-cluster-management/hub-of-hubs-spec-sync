// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package sync

import (
	"context"

	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/jackc/pgx/v4/pgxpool"
)

const controllerName string = "policy-spec-syncer"

var log = logf.Log.WithName(controllerName)

// Add creates a new Policy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, databaseConnectionPool *pgxpool.Pool) error {
	return add(mgr, newReconciler(mgr, databaseConnectionPool))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, databaseConnectionPool *pgxpool.Pool) reconcile.Reconciler {
	return &ReconcilePolicy{client: mgr.GetClient(), scheme: mgr.GetScheme(), databaseConnectionPool: databaseConnectionPool}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Policy
	err = c.Watch(&source.Kind{Type: &policiesv1.Policy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
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

	// Fetch the Policy instance
	instance := &policiesv1.Policy{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)

	if err != nil {
		if errors.IsNotFound(err) {
			// repliated policy on hub was deleted, update all the matching policies in the database as deleted
			reqLogger.Info("Policy was deleted, update the deleted field in the database...")

			_, err = r.databaseConnectionPool.Exec(context.Background(),
				`UPDATE spec.policies SET deleted = true WHERE payload -> 'metadata' ->> 'name' = $1 AND
			     payload -> 'metadata' ->> 'namespace' = $2`, request.Name, request.Namespace)

			if err != nil {
				log.Error(err, "Delete failed")
				return reconcile.Result{}, err
			}

			reqLogger.Info("Policy has been updated as deleted in the database...Reconciliation complete.")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get policy from hub...")
		return reconcile.Result{}, err
	}

	_, err = r.databaseConnectionPool.Exec(context.Background(),
		"INSERT INTO spec.policies (id,payload) values($1, $2::jsonb)", string(instance.UID), &instance)
	if err != nil {
		log.Error(err, "Insert failed")
	}

	rows, err := r.databaseConnectionPool.Query(context.Background(), "select payload -> 'metadata' -> 'name' as name from spec.policies")
	if err != nil {
		log.Error(err, "Query failed")
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		rows.Scan(&name)
		log.Info(name)
	}

	return reconcile.Result{}, err
}
