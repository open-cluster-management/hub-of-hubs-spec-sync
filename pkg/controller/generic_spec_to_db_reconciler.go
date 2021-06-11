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

	instance := r.createInstance()
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// the instance on hub was deleted, update all the matching instances in the database as deleted
			err = r.deleteFromTheDatabase(request.Name, request.Namespace)
			if err != nil {
				reqLogger.Error(err, "Delete failed")
				return ctrl.Result{}, err
			}

			reqLogger.Info("Reconciliation complete.")

			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get the instance from hub...")

		return ctrl.Result{}, err
	}

	instanceID := instance.GetUID()

	if instance.GetDeletionTimestamp().IsZero() {
		if !containsString(instance.GetFinalizers(), r.finalizerName) {
			reqLogger.Info("Adding finalizer")
			controllerutil.AddFinalizer(instance, r.finalizerName)
			if err := r.client.Update(ctx, instance); err != nil {
				return ctrl.Result{Requeue: true, RequeueAfter: requeuePeriodSeconds * time.Second}, err
			}
		}
	} else {
		if containsString(instance.GetFinalizers(), r.finalizerName) {
			// the policy is being deleted, update all the matching policies in the database as deleted
			if err := r.deleteFromTheDatabase(request.Name, request.Namespace); err != nil {
				reqLogger.Error(err, "Delete failed")
				return ctrl.Result{}, err
			}
			reqLogger.Info("Removing finalizer")
			controllerutil.RemoveFinalizer(instance, r.finalizerName)
			if err = r.client.Update(ctx, instance); err != nil {
				return ctrl.Result{Requeue: true, RequeueAfter: requeuePeriodSeconds * time.Second}, err
			}
		}
		reqLogger.Info("Reconciliation complete.")
		return ctrl.Result{}, nil
	}

	cleanInstance(instance)

	instanceInTheDatabase := r.createInstance()
	err = r.databaseConnectionPool.QueryRow(ctx,
		fmt.Sprintf("SELECT payload FROM spec.%s WHERE id = $1", r.tableName),
		string(instanceID)).Scan(&instanceInTheDatabase)

	if errors.Is(err, pgx.ErrNoRows) {
		reqLogger.Info("The instance with the current UID does not exist in the database, inserting...")
		_, err := r.databaseConnectionPool.Exec(ctx,
			fmt.Sprintf("INSERT INTO spec.%s (id,payload) values($1, $2::jsonb)", r.tableName),
			string(instanceID), &instance)
		if err != nil {
			reqLogger.Error(err, "Insert failed")
		} else {
			reqLogger.Info("The instance has been inserted into the database...Reconciliation complete.")
		}

		return ctrl.Result{}, err
	}

	if !r.areEqual(instance, instanceInTheDatabase) {
		reqLogger.Info("Mismatch between hub and the database, updating the database...")

		_, err := r.databaseConnectionPool.Exec(ctx,
			fmt.Sprintf("UPDATE spec.%s SET payload = $1 WHERE id = $2", r.tableName),
			&instance, string(instanceID))
		if err != nil {
			reqLogger.Error(err, "Update failed")
			return ctrl.Result{}, err
		}
	}

	reqLogger.Info("Reconciliation complete.")

	return ctrl.Result{}, err
}

func cleanInstance(instance object) {
	instance.SetUID("")
	instance.SetResourceVersion("")
	instance.SetManagedFields(nil)
	instance.SetFinalizers(nil)
	instance.SetGeneration(0)
	instance.SetOwnerReferences(nil)
	instance.SetClusterName("")

	delete(instance.GetAnnotations(), "kubectl.kubernetes.io/last-applied-configuration")
}

func (r *genericSpecToDBReconciler) deleteFromTheDatabase(name, namespace string) error {
	reqLogger := r.log.WithValues("Request.Namespace", namespace, "Request.Name", name)
	// the policy on hub was deleted, update all the matching policies in the database as deleted
	reqLogger.Info("Instance was deleted, update the deleted field in the database...")

	_, err := r.databaseConnectionPool.Exec(context.Background(),
		fmt.Sprintf(`UPDATE spec.%s SET deleted = true WHERE payload -> 'metadata' ->> 'name' = $1 AND
			     payload -> 'metadata' ->> 'namespace' = $2 AND deleted = false`, r.tableName), name, namespace)

	if err == nil {
		reqLogger.Info("Instance has been updated as deleted in the database...")
	}

	return err
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
