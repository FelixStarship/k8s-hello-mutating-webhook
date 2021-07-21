package api

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/gofrs/uuid"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os"
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

	registry := os.Getenv("registry")

	for i := 0; i < len(deploy.Spec.Template.Spec.Containers); i++ {
		deploy.Spec.Template.Spec.Containers[i].Image = fmt.Sprint(registry, "/", GenerateImageUrl(deploy.Spec.Template.Spec.Containers[i].Image))
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

func GetMd5Prefix(str string) string {
	data := []byte(str)
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has)
	return md5str
}

// generateImageUrl 生成镜像url
// 返回的名称格式：hash + registry + namespace + repository:tag
// hash        : registry + namespace的md5的前8位
// registry + namespace : 此部分为可选字段，如果有剩余的字段，则会填充此部分
// repository:tag :imageUrl原来的repository。如果长度超过55，那么将会修改repository:tag。修改规则为 “：左边取34个字符，右边取20个字符”
func GenerateImageUrl(imageUrl string) string {

	var registry = "registry.hub.docker.com"
	var namespace = "library"
	var repositoryAndTag string

	if imageUrl == "" {
		return ""
	}
	fileds := strings.Split(imageUrl, "/")

	// 获取registry
	if len(fileds) >= 3 {
		registry = fileds[len(fileds)-3]
	}
	if index := strings.LastIndexByte(registry, '.'); index > 0 { // 去掉最后的 .com .io 等
		registry = registry[:index]
	}
	registry = strings.Trim(registry, ".") // 去掉所有的 .

	// 获取namespace
	if len(fileds) >= 2 {
		namespace = fileds[len(fileds)-2]
	}

	// 获取repository
	if len(fileds) >= 1 {
		repositoryAndTag = fileds[len(fileds)-1]
	}

	// 创建名称
	if len(repositoryAndTag) > 55 { // 阿里云的长度限制为 64
		fileds := strings.Split(repositoryAndTag, ":")
		if len(fileds) == 2 {
			tagLen := len(fileds[1])
			if tagLen > 20 {
				tagLen = 20
			}
			tag := fileds[1][:tagLen]

			repositoryLen := 55 - 1 - len(tag)
			if repositoryLen > len(fileds[0]) {
				repositoryLen = len(fileds[0])
			}

			repositoryAndTag = fileds[0][:repositoryLen] + ":" + tag
		} else {
			repositoryAndTag = repositoryAndTag[0:55]
		}
	}

	// 创建前缀
	prefix := GetMd5Prefix(registry + namespace)[:6]

	// 创建中间段
	if len(repositoryAndTag)+len(prefix) < 64-2 { // 中间有两个下划线
		namespace = registry + namespace
		namespaceMaxLen := 64 - 2 - len(repositoryAndTag) - len(prefix)
		if namespaceMaxLen > len(namespace) {
			namespaceMaxLen = len(namespace)
		}
		if namespaceMaxLen > 0 {
			namespace = namespace[:namespaceMaxLen]
		}
	}

	return fmt.Sprintf("%v_%v_%v", prefix, namespace, repositoryAndTag)
}
