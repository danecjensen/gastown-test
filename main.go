package main

import (
	"net/http"
	"os/exec"
	"sync"

	"github.com/gin-gonic/gin"
)

// ScriptRegistry holds the mapping of routes to script paths
type ScriptRegistry struct {
	mu      sync.RWMutex
	scripts map[string]string // route path -> script path
}

var registry = &ScriptRegistry{
	scripts: make(map[string]string),
}

// Register adds a new route-to-script mapping
func (r *ScriptRegistry) Register(route, scriptPath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scripts[route] = scriptPath
}

// Get retrieves the script path for a route
func (r *ScriptRegistry) Get(route string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	script, ok := r.scripts[route]
	return script, ok
}

// List returns all registered routes
func (r *ScriptRegistry) List() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range r.scripts {
		result[k] = v
	}
	return result
}

// Delete removes a route mapping
func (r *ScriptRegistry) Delete(route string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.scripts[route]; ok {
		delete(r.scripts, route)
		return true
	}
	return false
}

// executeScript runs a script and returns its output
func executeScript(scriptPath string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", scriptPath)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func main() {
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Welcome to the Spinride API"})
	})

	// Script management API
	scripts := r.Group("/api/v1/scripts")
	{
		// List all registered scripts
		scripts.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"scripts": registry.List()})
		})

		// Register a new script route
		scripts.POST("", func(c *gin.Context) {
			var req struct {
				Route      string `json:"route" binding:"required"`
				ScriptPath string `json:"script_path" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			registry.Register(req.Route, req.ScriptPath)
			c.JSON(http.StatusCreated, gin.H{
				"message":     "Script registered",
				"route":       req.Route,
				"script_path": req.ScriptPath,
			})
		})

		// Delete a script route
		scripts.DELETE("/:route", func(c *gin.Context) {
			route := "/" + c.Param("route")
			if registry.Delete(route) {
				c.JSON(http.StatusOK, gin.H{"message": "Script route deleted", "route": route})
			} else {
				c.JSON(http.StatusNotFound, gin.H{"error": "Route not found"})
			}
		})
	}

	// Dynamic script execution endpoint
	r.GET("/exec/*route", func(c *gin.Context) {
		route := c.Param("route")
		scriptPath, ok := registry.Get(route)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "No script registered for route: " + route})
			return
		}

		output, err := executeScript(scriptPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  "Script execution failed",
				"detail": err.Error(),
				"output": output,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"route":  route,
			"output": output,
		})
	})

	// POST version for scripts that need input
	r.POST("/exec/*route", func(c *gin.Context) {
		route := c.Param("route")
		scriptPath, ok := registry.Get(route)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "No script registered for route: " + route})
			return
		}

		// Get optional stdin from request body
		var req struct {
			Stdin string `json:"stdin"`
		}
		c.ShouldBindJSON(&req)

		var cmd *exec.Cmd
		if req.Stdin != "" {
			cmd = exec.Command("/bin/bash", "-c", scriptPath)
			stdin, _ := cmd.StdinPipe()
			go func() {
				defer stdin.Close()
				stdin.Write([]byte(req.Stdin))
			}()
		} else {
			cmd = exec.Command("/bin/bash", "-c", scriptPath)
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  "Script execution failed",
				"detail": err.Error(),
				"output": string(output),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"route":  route,
			"output": string(output),
		})
	})

	r.Run(":8080")
}
