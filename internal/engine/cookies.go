package engine

import (
	"net/http"
	"net/http/cookiejar"
)

type CookieManager struct {
	jars map[string]*cookiejar.Jar
}

func NewCookieManager() *CookieManager {
	return &CookieManager{jars: make(map[string]*cookiejar.Jar)}
}

func (cm *CookieManager) JarForEnv(envName string) http.CookieJar {
	if jar, ok := cm.jars[envName]; ok {
		return jar
	}
	jar, _ := cookiejar.New(nil)
	cm.jars[envName] = jar
	return jar
}

func (cm *CookieManager) ClearEnv(envName string) {
	delete(cm.jars, envName)
}

func (cm *CookieManager) ClearAll() {
	cm.jars = make(map[string]*cookiejar.Jar)
}
