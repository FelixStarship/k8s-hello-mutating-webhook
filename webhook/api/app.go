package api

import (
	"encoding/json"
	"fmt"
	"github.com/gofrs/uuid"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os/exec"
	"strings"
)

type App struct {
}

func (app *App) HandleMutate(w http.ResponseWriter, r *http.Request) {
	admissionReview := &admissionv1.AdmissionReview{}

	// read the AdmissionReview from the request json body
	err := readJSON(r, admissionReview)
	if err != nil {
		app.HandleError(w, r, err)
		return
	}

	// unmarshal the pod from the AdmissionRequest
	deploy := &appsv1.Deployment{}
	if err := json.Unmarshal(admissionReview.Request.Object.Raw, deploy); err != nil {
		app.HandleError(w, r, fmt.Errorf("unmarshal to pod: %v", err))
		return
	}

	uid, _ := uuid.NewV4()
	err = Run(deploy.Namespace, uid.String())

	if err != nil {
		app.HandleError(w, r, err)
		return
	}

	// add the imagePullSecrets to the pod
	deploy.Spec.Template.Spec.ImagePullSecrets = append(deploy.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{
		Name: uid.String(),
	})

	labels, err := ImageFilter(deploy.Labels)
	if err != nil {
		app.HandleError(w, r, err)
		return
	}

	for i := 0; i < len(deploy.Spec.Template.Spec.Containers); i++ {
		imagesArgs := strings.Split(deploy.Spec.Template.Spec.Containers[i].Image, "/")
		deploy.Spec.Template.Spec.Containers[i].Image = fmt.Sprint(labels, "/", imagesArgs[1], "/", imagesArgs[2])
		//deploy.Spec.Template.Spec.Containers[i].Image="docker-prod-registry.cn-hangzhou.cr.aliyuncs.com/cloudnative/test:202107131832"
	}

	containersBytes, err := json.Marshal(&deploy.Spec.Template.Spec.Containers)
	if err != nil {
		app.HandleError(w, r, fmt.Errorf("marshall containers: %v", err))
		return
	}

	secretsBytes, err := json.Marshal(&deploy.Spec.Template.Spec.ImagePullSecrets)
	if err != nil {
		app.HandleError(w, r, fmt.Errorf("marshall volumes: %v", err))
		return
	}

	// build json patch
	patch := []JSONPatchEntry{
		{
			OP:    "replace",
			Path:  "/spec/template/spec/containers",
			Value: containersBytes,
		},
		{
			OP:    "replace",
			Path:  "/spec/template/spec/imagePullSecrets",
			Value: secretsBytes,
		},
	}

	patchBytes, err := json.Marshal(&patch)
	if err != nil {
		app.HandleError(w, r, fmt.Errorf("marshall jsonpatch: %v", err))
		return
	}

	patchType := admissionv1.PatchTypeJSONPatch

	// build admission response
	admissionResponse := &admissionv1.AdmissionResponse{
		UID:       admissionReview.Request.UID,
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &patchType,
	}

	respAdmissionReview := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: admissionResponse,
	}

	jsonOk(w, &respAdmissionReview)
}

type JSONPatchEntry struct {
	OP    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
}

func Run(namespace, secretName string) error {

	//var secret=new(corev1.Secret)
	//var err error
	//
	//client,err:=NewClientSet()
	//if err!=nil {
	//	return err
	//}
	//
	//secret,err= client.CoreV1().Secrets(namespace).Get(context.TODO(),secretName,metav1.GetOptions{})
	//
	//if secret!=nil&&secret.Name=="" {
	//
	//	secret,err=client.CoreV1().Secrets("devops").Create(context.TODO(),&corev1.Secret{
	//		TypeMeta:metav1.TypeMeta{
	//			Kind: "Secret",
	//			APIVersion: "v1",
	//		},
	//		ObjectMeta:metav1.ObjectMeta{
	//			Name: secretName,
	//			Namespace: namespace,
	//		},
	//		Type: "kubernetes.io/dockerconfigjson",
	//		Data: map[string][]byte{
	//			".dockercfg":[]byte("{\"auths\": {\"docker-prod-registry.cn-hangzhou.cr.aliyuncs.com/cloudnative\": {\"username\": \"mysoft_paas\", \"password\": \"Mypaas@2020\", \"auth\": \"bXlzb2Z0X3BhYXM6TXlwYWFzQDIwMjA=\"}}}"),
	//		},
	//	},metav1.CreateOptions{})
	//}
	//return err
	var args []string
	args = append(args, "kubectl create secret docker-registry ", secretName, " ")
	args = append(args, "--docker-server=docker-prod-registry.cn-hangzhou.cr.aliyuncs.com/cloudnative ")
	args = append(args, "--docker-username='mysoft_paas' ")
	args = append(args, "--docker-password='Mypaas@2020' ")
	args = append(args, "--namespace=", namespace)

	cmd := strings.Join(args, "")

	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()

	if err != nil {
		return fmt.Errorf("Failed to run cmd: " + cmd + ", with out: " + string(out) + ", with error: " + err.Error())
	}
	return nil
}

func ImageFilter(labels map[string]string) (string, error) {
	if registry, ok := labels["registry"]; ok {
		return registry, nil
	}
	return "", fmt.Errorf("labels registry required parameters ")
}

//func NewClientSet() (*kubernetes.Clientset, error) {
//	kubeConfigLocation := filepath.Join(os.Getenv("HOME"), ".kube", "config")
//	if _, err := os.Stat(kubeConfigLocation); err != nil {
//		kubeConfigLocation = ""
//	}
//	// use the current context in kubeconfig
//	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigLocation)
//	if err != nil {
//		return nil, err
//	}
//	return kubernetes.NewForConfig(config)
//}
