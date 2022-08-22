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

package v2alpha1

import (
	"errors"
	"fmt"

	emperror "emperror.dev/errors"
	"github.com/gurkankaymak/hocon"
	"github.com/sethvargo/go-password/password"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var emqxlog = logf.Log.WithName("emqx-resource")

func (r *EMQX) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-apps-emqx-io-v2alpha1-emqx,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxes,verbs=create;update,versions=v2alpha1,name=mutating.apps.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &EMQX{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EMQX) Default() {
	emqxlog.Info("default", "name", r.Name)

	bootstrapConfig, err := defaultBootstrapConfig(r.Spec.BootstrapConfig)
	if err != nil {
		emqxlog.Error(err, "default bootstrap config failed")
		return
	}
	r.Spec.BootstrapConfig = bootstrapConfig

	// Labels
	if r.Labels == nil {
		r.Labels = make(map[string]string)
	}
	r.Labels["apps.emqx.io/managed-by"] = "emqx-operator"
	r.Labels["apps.emqx.io/instance"] = r.GetName()

	if r.Spec.CoreTemplate.Labels == nil {
		r.Spec.CoreTemplate.Labels = make(map[string]string)
	}
	r.Spec.CoreTemplate.Labels["apps.emqx.io/instance"] = r.Name
	r.Spec.CoreTemplate.Labels["apps.emqx.io/managed-by"] = "emqx-operator"
	r.Spec.CoreTemplate.Labels["apps.emqx.io/db-role"] = "core"

	// Replicant
	if r.Spec.ReplicantTemplate.Labels == nil {
		r.Spec.ReplicantTemplate.Labels = make(map[string]string)
	}
	r.Spec.ReplicantTemplate.Labels["apps.emqx.io/instance"] = r.Name
	r.Spec.ReplicantTemplate.Labels["apps.emqx.io/managed-by"] = "emqx-operator"
	r.Spec.ReplicantTemplate.Labels["apps.emqx.io/db-role"] = "replicant"

	// Replicant replicas
	defaultReplicas := int32(0)
	r.Spec.ReplicantTemplate.Spec.Replicas = &defaultReplicas
}

func defaultBootstrapConfig(bootstrapConfig string) (string, error) {
	password, _ := password.Generate(64, 10, 0, true, true)
	defaultBootstrapConfigStr := fmt.Sprintf(`
	node {
	  cookie = "%s"
	  data_dir = "data"
	  etc_dir = "etc"
	}
	dashboard {
	  listeners.http {
		bind: "18083"
	  }
	  default_username: "admin"
	  default_password: "public"
	}
	listeners.tcp.default {
		bind = "0.0.0.0:1883"
		max_connections = 1024000
	}
	`, password)

	bootstrapConfig = fmt.Sprintf("%s\n%s", defaultBootstrapConfigStr, bootstrapConfig)
	config, err := hocon.ParseString(bootstrapConfig)
	if err != nil {
		return "", err
	}
	return config.String(), nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-apps-emqx-io-v2alpha1-emqx,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxes,verbs=create;update,versions=v2alpha1,name=validator.apps.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &EMQX{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EMQX) ValidateCreate() error {
	emqxlog.Info("validate create", "name", r.Name)

	if _, err := hocon.ParseString(r.Spec.BootstrapConfig); err != nil {
		err = emperror.Wrap(err, "failed to parse bootstrap config")
		emqxlog.Error(err, "validate create failed")
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EMQX) ValidateUpdate(old runtime.Object) error {
	emqxlog.Info("validate update", "name", r.Name)

	oldEMQX := old.(*EMQX)
	if r.Spec.BootstrapConfig != oldEMQX.Spec.BootstrapConfig {
		emqxlog.Info("validate update", "name", r.Name, "old bootstrap config", oldEMQX.Spec.BootstrapConfig, "new bootstrap config", r.Spec.BootstrapConfig)
		err := errors.New("bootstrap config cannot be updated")
		emqxlog.Error(err, "validate update failed")
		return err
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EMQX) ValidateDelete() error {
	emqxlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}