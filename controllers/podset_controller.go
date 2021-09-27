/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/appconfig"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	//"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mydomainv1alpha1 "podset-operator/api/v1alpha1"
	"time"
)

// PodSetReconciler reconciles a PodSet object
type PodSetReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	Region       string
	RequeueAfter time.Duration
}

//+kubebuilder:rbac:groups=my.domain,resources=podsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=my.domain,resources=podsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my.domain,resources=podsets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PodSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *PodSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//ctx := context.Background()

	requeue30 := ctrl.Result{RequeueAfter: 30 * time.Second}
	//requeue5 := ctrl.Result{RequeueAfter: 5 * time.Second}
	requeue := ctrl.Result{Requeue: true}
	//forget := ctrl.Result{}

	var PodSet mydomainv1alpha1.PodSet
	var result map[string]interface{}

	if err := r.Get(ctx, req.NamespacedName, &PodSet); err != nil {
		ctrl.Log.Error(err, "unable to fetch PodSet")

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.Region = "eu-west-1"

	//fmt.Println(PodSet.Spec.Labels)
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(r.Region)},
	)

	if err != nil {
		fmt.Println(err)
	}

	svc := appconfig.New(sess)

	config, err := svc.GetConfiguration(&appconfig.GetConfigurationInput{
		Application:                &PodSet.Spec.Application,
		ClientId:                   &PodSet.Spec.ClientID,
		ClientConfigurationVersion: &PodSet.Spec.ClientConfigurationVersion,
		Configuration:              &PodSet.Spec.Configuration,
		Environment:                &PodSet.Spec.Environment,
	})

	if err != nil {
		fmt.Println(err)
		return requeue, nil
	}

	fmt.Println(config)
	fmt.Println(*config.ConfigurationVersion)
	fmt.Println(PodSet.Spec.ClientConfigurationVersion)
	fmt.Println(PodSet.Spec.Labels)

	config_version := json.Unmarshal([]byte(*config.ConfigurationVersion), &result)
	if config_version != nil {
		fmt.Println("Error", err)
	}

	fmt.Println(result)

	config_content := json.Unmarshal([]byte(config.Content), &result)
	if config_content != nil {
		fmt.Println("Error", err)
	}

	fmt.Println(result)

	version := *config.ConfigurationVersion

	if version != PodSet.Spec.ClientConfigurationVersion {
		fmt.Println("continuing to next loop")

		var deploy v1.DeploymentList
		r.List(ctx, &deploy, client.MatchingLabels(PodSet.Spec.Labels))
		fmt.Println("List deployments by Label:", deploy)

		for _, deployment := range deploy.Items {
			// Patch the Deployment with new label containing redeployed timestamp, to force redeploy
			fmt.Println("Rotating deployment", deployment.ObjectMeta.Name)
			patch := []byte(fmt.Sprintf(`{"spec":{"template":{"metadata":{"labels":{"aws-secrets-operator-redeloyed":"%v"}}}}}`, time.Now().Unix()))
			if err := r.Patch(ctx, &deployment, client.RawPatch(types.StrategicMergePatchType, patch)); err != nil {
				fmt.Println("Patch deployment err:", err)
				return requeue30, nil
			}
		}

	}

	return requeue30, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *PodSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mydomainv1alpha1.PodSet{}).
		Complete(r)
}

