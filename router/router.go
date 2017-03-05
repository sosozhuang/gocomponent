package router

import (
	"gopkg.in/macaron.v1"
	"github.com/sosozhuang/component/handler"
)

func SetRouters(m *macaron.Macaron) {
	m.Group("/v2", func() {
		m.Get("/", handler.IndexHandler)

		//todo: remove begin
		m.Post("/test", handler.TestHandler)
		//todo: remove end

		m.Group("/events", func() {
			m.Post("/", handler.CreateEvent)
		})

		m.Group("/components", func() {
			m.Get("/", handler.ListComponents)
			m.Post("/", handler.CreateComponent)

			m.Post("/:component", handler.SaveComponentAsNewVersion)
			m.Get("/:component", handler.GetComponent)
			m.Put("/:component", handler.UpdateComponent)
			m.Delete("/:component", handler.DeleteComponent)

			m.Get("/:component/debug", handler.DebugComponentJson(), handler.DebugComponent)
			m.Post("/:component/execute", handler.StartComponent)
		})

		m.Group("/executions", func() {
			m.Get("/:execution", handler.GetComponentExecution)
			m.Delete("/:execution", handler.StopComponentExecution)
		})

		m.Group("/images", func() {
			m.Post("/check", handler.CheckImageScript)
			//todo: remove begin
			m.Post("/build", handler.BuildImage)
			m.Post("/test", handler.TestHandler)
			//todo: remove end
		})
	})
}
