package api

import (
	"fmt"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

func InitLogger() zap.SugaredLogger {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"} // write to stdout
	config.InitialFields = map[string]interface{}{}
	logger, configErr := config.Build()
	if configErr != nil {
		fmt.Printf("Error while initializing the logger: %v", configErr)
		panic(configErr)
	}
	return *logger.Sugar()
}

// GetConfig returns a Kubernetes config
func GetConfig() (*rest.Config, error) {
	var config *rest.Config

	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if _, err := os.Stat(kubeconfig); err == nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// Contains checks if a string is present in a slice of strings
func Contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
		}
		return true
	}
	return false
}
