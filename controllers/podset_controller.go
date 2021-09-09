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
	"github.com/aws/aws-sdk-go"
	"github.com/aws/aws-sdk-go/service/appconfig"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mydomainv1alpha1 "podset-operator/api/v1alpha1"
	"time"
)

// PodSetReconciler reconciles a PodSet object
type PodSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Region       string
	Clock
	Application string 
	ClientConfigurationVersion int32
	Configuration string
	Environment string
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
func (r *PodSetReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	var PodSet mydomainv1alpha1.PodSet
	var result map[string]interface{}

	if err := r.Get(ctx, req.NamespacedName, &PodSet); err != nil {
		log.Error(err, "unable to fetch PodSet")

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	fmt.Println(PodSet.Spec.Labels)
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(r.Region)},
	)

	svc := appconfig.New(sess)

	
	for ;; {
		
		config, err := svc.GetConfiguration(&appconfig.GetConfigurationInput{
			Application: &r.Application,
			ClientConfigurationVersion: &r.ClientConfigurationVersion,
			Configuration: &r.Configuration,
			Environment: &r.Environment,
			//To change if in status
		})

		
		
		err := json.Unmarshal([]byte(*config.Body), &result)
			if err != nil {
				fmt.Println("Error", err)
			}
		version :=result["Configuration-Version"].(map[string]interface{})

		if version != SecretsRotationMapping.Spec.VersionID {
			fmt.Println("continuing to next loop")
			continue
		}

		

		var deploy v1.DeploymentList
			//MatchingLabels := SecretsRotationMapping.Spec.Labels
			r.List(ctx, &deploy, client.MatchingLabels(PodSet.Spec.Labels))
			
			//	fmt.Println("List deployments by Label:", deploy)

			for _, deployment := range deploy.Items {
				// Patch the Deployment with new label containing redeployed timestamp, to force redeploy
				fmt.Println("Rotating deployment", deployment.ObjectMeta.Name)
				patch := []byte(fmt.Sprintf(`{"spec":{"template":{"metadata":{"labels":{"aws-controller-redeployed":"%v"}}}}}`, time.Now().Unix()))
				if err := r.Patch(ctx, &deployment, client.RawPatch(types.StrategicMergePatchType, patch)); err != nil {
					fmt.Println("Patch deployment err:", err)
					return ctrl.Result{RequeueAfter: time.Second * r.RequeueAfter}, nil
				}
			}




		time.Sleep(30 * time.Minute)
		// Add the polling
	}
	

	if err != nil {
		fmt.Println("Error", err)
		return ctrl.Result{RequeueAfter: time.Second * r.RequeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mydomainv1alpha1.PodSet{}).
		Owns(&mydomainv1alpha1.Deployment{}).
		Complete(r)
}
