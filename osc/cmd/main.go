package main

import "github.com/hypebeast/go-osc/osc"

func main() {
	addr := "127.0.0.1:9191"
	d := osc.NewStandardDispatcher()
	d.AddMsgHandler("/play", func(msg *osc.Message) {
		osc.PrintMessage(msg)
	})

	server := &osc.Server{
		Addr:       addr,
		Dispatcher: d,
	}
  if err := server.ListenAndServe(); err != nil {
    panic(err)
  }
}
