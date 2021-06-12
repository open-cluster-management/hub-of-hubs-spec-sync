// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	pgx "github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type genericSpecToDBReconciler struct {
	client                 client.Client
	log                    logr.Logger
	databaseConnectionPool *pgxpool.Pool
	tableName              string
	finalizerName          string
	createInstance         func() object
	areEqual               func(instance1, instance2 object) bool
}

type object interface {
	metav1.Object
	runtime.Object
}

const requeuePeriodSeconds = 5

func (r *genericSpecToDBReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info(fmt.Sprintf("Reconciling %s ...", r.tableName))

	ctx := context.Background()

	instance, err := r.processCR(ctx, request, reqLogger)
	if err != nil {
		reqLogger.Info("Reconciliation failed: %v", err)
		return ctrl.Result{Requeue: true, RequeueAfter: requeuePeriodSeconds * time.Second}, err
	}

	if instance == nil {
		reqLogger.Info("Reconciliation complete.")
		return ctrl.Result{}, nil
	}

	instanceInTheDatabase, err := r.processInstanceInTheDatabase(ctx, instance, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Reconciliation failed")
		return ctrl.Result{Requeue: true, RequeueAfter: requeuePeriodSeconds * time.Second}, err
	}

	if !r.areEqual(instance, instanceInTheDatabase) {
		reqLogger.Info("Mismatch between hub and the database, updating the database")

		_, err := r.databaseConnectionPool.Exec(ctx,
			fmt.Sprintf("UPDATE spec.%s SET payload = $1 WHERE id = $2", r.tableName),
			&instance, string(instance.GetUID()))
		if err != nil {
			err = fmt.Errorf("failed to update the database with new value: %w", err)
			reqLogger.Error(err, "Reconciliation failed")

			return ctrl.Result{}, err
		}
	}

	reqLogger.Info("Reconciliation complete.")

	return ctrl.Result{}, err
}

func (r *genericSpecToDBReconciler) processCR(ctx context.Context, request ctrl.Request,
	log logr.Logger) (object, error) {
	instance := r.createInstance()

	err := r.client.Get(ctx, request.NamespacedName, instance)
	if apierrors.IsNotFound(err) {
		// the instance on hub was deleted, update all the matching instances in the database as deleted
		return nil, r.deleteFromTheDatabase(request.Name, request.Namespace, log)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get the instance from hub: %w", err)
	}

	if isInstanceBeingDeleted(instance) {
		return nil, r.removeFinalizerAndDelete(ctx, instance, log)
	}

	return cleanInstance(instance), r.addFinalizer(ctx, instance, log)
}

func isInstanceBeingDeleted(instance object) bool {
	return !instance.GetDeletionTimestamp().IsZero()
}

func (r *genericSpecToDBReconciler) removeFinalizerAndDelete(ctx context.Context, instance object,
	log logr.Logger) error {
	if !containsString(instance.GetFinalizers(), r.finalizerName) {
		return nil
	}

	log.Info("Removing an instance from the database")

	// the policy is being deleted, update all the matching policies in the database as deleted
	if err := r.deleteFromTheDatabase(instance.GetName(), instance.GetNamespace(), log); err != nil {
		return fmt.Errorf("failed to delete an instance from the database: %w", err)
	}

	log.Info("Removing finalizer")
	controllerutil.RemoveFinalizer(instance, r.finalizerName)

	if err := r.client.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to remove a finalizer: %w", err)
	}

	return nil
}

func (r *genericSpecToDBReconciler) addFinalizer(ctx context.Context, instance object, log logr.Logger) error {
	if containsString(instance.GetFinalizers(), r.finalizerName) {
		return nil
	}

	log.Info("Adding finalizer")
	controllerutil.AddFinalizer(instance, r.finalizerName)

	if err := r.client.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to add a finalizer: %w", err)
	}

	return nil
}

func (r *genericSpecToDBReconciler) processInstanceInTheDatabase(ctx context.Context, instance object,
	log logr.Logger) (object, error) {
	instanceInTheDatabase := r.createInstance()
	err := r.databaseConnectionPool.QueryRow(ctx,
		fmt.Sprintf("SELECT payload FROM spec.%s WHERE id = $1", r.tableName),
		string(instance.GetUID())).Scan(&instanceInTheDatabase)

	if errors.Is(err, pgx.ErrNoRows) {
		log.Info("The instance with the current UID does not exist in the database, inserting...")

		_, err := r.databaseConnectionPool.Exec(ctx,
			fmt.Sprintf("INSERT INTO spec.%s (id,payload) values($1, $2::jsonb)", r.tableName),
			string(instance.GetUID()), &instance)
		if err != nil {
			return nil, fmt.Errorf("insert into database failed: %w", err)
		}

		log.Info("The instance has been inserted into the database")

		return instance, nil // the instance in the database is identical to the instance we just inserted
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get the instance in the database: %w", err)
	}

	return instanceInTheDatabase, nil
}

func cleanInstance(instance object) object {
	instance.SetUID("")
	instance.SetResourceVersion("")
	instance.SetManagedFields(nil)
	instance.SetFinalizers(nil)
	instance.SetGeneration(0)
	instance.SetOwnerReferences(nil)
	instance.SetClusterName("")

	delete(instance.GetAnnotations(), "kubectl.kubernetes.io/last-applied-configuration")

	return instance
}

func (r *genericSpecToDBReconciler) deleteFromTheDatabase(name, namespace string, log logr.Logger) error {
	log.Info("Instance was deleted, update the deleted field in the database")

	_, err := r.databaseConnectionPool.Exec(context.Background(),
		fmt.Sprintf(`UPDATE spec.%s SET deleted = true WHERE payload -> 'metadata' ->> 'name' = $1 AND
			     payload -> 'metadata' ->> 'namespace' = $2 AND deleted = false`, r.tableName), name, namespace)
	if err != nil {
		return fmt.Errorf("failed to delete instance from the database: %w", err)
	}

	log.Info("Instance has been updated as deleted in the database")

	return nil
}

// from https://book.kubebuilder.io/reference/using-finalizers.html
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}

	return false
}
