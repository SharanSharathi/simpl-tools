package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

var stopChan chan os.Signal

func init() {
	stopChan = make(chan os.Signal, 1)
	go stopServer()

	signal.Notify(stopChan, syscall.SIGINT)

	flag.IntVar(&port, "port", 8000, "server port")
	flag.BoolVar(&noEnd, "noend", false, "remove /end path")
}

var (
	port  int
	noEnd bool
)

func main() {
	flag.Parse()

	args := flag.Args()

	if len(args) == 0 {
		fmt.Println("pass some files or directory to serve!")
		return
	}

	if !noEnd {
		addServerEndPath("/end")
	}

	for _, item := range args {
		err := serveItem(item)
		if err != nil {
			fmt.Println()
			fmt.Println(err)
			return
		}
	}

	startServer(port)
}

func serveItem(item string) (err error) {
	item, err = filepath.Abs(item)
	if err != nil {
		return
	}

	fi, err := os.Stat(item)
	if err != nil {
		return err
	}

	defer func() {
		e := recover()
		if e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()

	var pattern string = "/" + fi.Name()

	fmt.Println("item: " + item)
	fmt.Printf("path (%s): ", pattern)
	fmt.Scanln(&pattern)

	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	if fi.IsDir() {
		if !strings.HasSuffix(pattern, "/") {
			pattern = pattern + "/"
		}

		serveDir(item, pattern)
	} else {
		serveFile(item, pattern)
	}

	return
}

func serveFile(file string, pattern string) {
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, file)
	})
}

func serveDir(dir string, pattern string) {
	http.Handle(pattern, http.StripPrefix(pattern, http.FileServer(http.Dir(dir))))
}

func startServer(port int) {
	fmt.Println("\nStarting Server...")
	fmt.Printf("http://%s:%d\n", getIP(), port)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		fmt.Println(err)
	}
}

func stopServer() {
	<-stopChan
	fmt.Println("\nServer Stopped!")
	os.Exit(0)
}

func addServerEndPath(pattern string) {
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Stopping the Server!"))
		stopChan <- syscall.SIGINT
	})
}

func getIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "localhost"
}
