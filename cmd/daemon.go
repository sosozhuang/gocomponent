package cmd

import (
	"crypto/tls"
	"fmt"
	"github.com/sosozhuang/component/model"
	log "github.com/Sirupsen/logrus"
	"github.com/containerops/configure"
	"github.com/spf13/cobra"
	"gopkg.in/macaron.v1"
	"net/http"
	"strings"
	"net"
	"github.com/sosozhuang/component/web"
	"net/url"
	"github.com/sosozhuang/component/module"
)

var address string
var port int64
var listenMode string
//var serviceUrl string

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Web subcommand start component's REST API daemon.",
	Long:  ``,
}

var startDaemonCmd = &cobra.Command{
	Use:   "start",
	Short: "Start component's REST API daemon.",
	Long:  ``,
	Run:   startDeamon,
}

func init() {
	RootCmd.AddCommand(daemonCmd)

	// Add start subcommand
	daemonCmd.AddCommand(startDaemonCmd)
	startDaemonCmd.Flags().StringVarP(&address, "address", "a", "0.0.0.0", "http or https listen address.")
	startDaemonCmd.Flags().Int64VarP(&port, "port", "p", 80, "the port of http.")
	listenMode = configure.GetString("daemon.listenmode")
	initServiceUrl()
}

func initServiceUrl() {
	var u url.URL
	u.Scheme = listenMode
	addr := address
	//port := cmd.GetPort()
	if addr == "0.0.0.0" || addr == "127.0.0.1" {
		if addrs, err := net.InterfaceAddrs(); err != nil {
			log.Errorln("Component get network interfaces error:", err)
			addr = "127.0.0.1"
		} else {
			for _, a := range addrs {
				if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						addr = ipnet.IP.String()
						break
					}
				}
			}
		}
	}
	u.Host = fmt.Sprintf("%s:%d", addr, port)
	module.ServiceUrl = u.String()
}

func startDeamon(cmd *cobra.Command, args []string) {
	defer model.CloseDB()
	logFile := getLogFile(strings.TrimSpace(configure.GetString("log.file")),
		configure.GetBool("log.append"))
	log.SetOutput(logFile)
	defer logFile.Close()
	setLogLevel(strings.ToLower(configure.GetString("log.level")))

	m := macaron.New()

	// Set Macaron Web Middleware And Routers
	web.SetMacaron(m)

	switch listenMode {
	case "http":
		listenAddr := fmt.Sprintf("%s:%d", address, port)
		log.Debugln("Component is listening:", listenAddr)
		if err := http.ListenAndServe(listenAddr, m); err != nil {
			log.Errorf("startDeamon http server error: %v\n", err.Error())
			return
		}
	case "https":
		listenAddr := fmt.Sprintf("%s:%d", address, port)
		server := &http.Server{Addr: listenAddr, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS10}, Handler: m}
		if err := server.ListenAndServeTLS(configure.GetString("https.certfile"), configure.GetString("https.keyfile")); err != nil {
			log.Errorf("startDeamon https server error: %v\n", err.Error())
			return
		}
	default:
		log.Errorln("startDeamon can't listen at mode:", listenMode)
	}
}
