package main
import (
  "context"
	"fmt"
	"net/http"
	"time"
	"bufio"
	"os"
	"io"
	// "io/ioutil"  
  "log"
  "strings"
  "encoding/json"
  "strconv"
  "regexp"
  "encoding/binary" 
  "encoding/base64"
  // "crypto/md5"
  // "crypto/sha1"
  // "mime"
  // "path"
  "errors"
  "path/filepath"
  "os/exec"
  "runtime"
  "github.com/gorilla/mux"
  // "flag"
  // "runtime/pprof"
  // "os/signal"
  // "syscall"
)

var config map[string]interface{}
var rdfdirs map[string]string
var currentwd string
var cwd string
var pathSep int32
var logger *log.Logger
var fs_root http.Handler
var fs_data http.Handler
var srv *http.Server

var web_addr string
var web_err error
var web_pwd string = ""

var version string = "1.7.4"
// var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func IsFile(name string) bool {
  fi, err := os.Stat(name)
  if err != nil {
    return false
  }
  mode := fi.Mode()
  return mode.IsRegular()
}

func isDir(name string) (bool, error) {
  fi, err := os.Stat(name)
  if err != nil {
    return false, err
  }
  return fi.IsDir(), nil
  // mode := fi.Mode()
  // return mode.IsRegular()
}

func CreateDir(name string) error {
  is, err := isDir(name) 
  if is{
    return nil
  }
  err = os.MkdirAll(name, 0755)
  return err
}

func resp500(w http.ResponseWriter, err error){
  w.Header().Add("Content-Type", "text/plain")
  err_msg := fmt.Sprintf("%s", err)
  logger.Print(err_msg)
  w.WriteHeader(http.StatusInternalServerError)
  w.Write([]byte(err_msg))
}

func saveFileHandle(w http.ResponseWriter, r *http.Request){
  if checkPwd(w, r) == 0 {return}
  if r.Method == "POST" {
    filename := r.FormValue("filename")
    content := r.FormValue("content")
    path := filepath.Dir(filename)
    logger.Println(path)
    err := CreateDir(path)
    if err == nil {
      f, err := os.Create(filename)
      if err != nil {
        resp500(w, err)
        return
      }
      f.WriteString(content)
    }else{
      resp500(w, err)
    }
  }
}

func saveBinFileHandle(w http.ResponseWriter, r *http.Request){
  if checkPwd(w, r) == 0 {return}
  w.Header().Add("Content-Type", "text/plain")
  file, _, _ := r.FormFile("file") 
  defer file.Close()
  // fW, err := os.Create("" + head.Filename)
  // logger.Printf("%q", head)
  filename := r.FormValue("filename")
  downloadpath := filepath.Dir(filename)
  err := CreateDir(downloadpath)
  if err == nil {
    fW, err := os.Create(filename)
    if err != nil {
      resp500(w, err)
      return
    }
    defer fW.Close()
    _, err = io.Copy(fW, file)
    if err != nil {
      resp500(w, err)
    }
    // logger.Println("File saved successful")
  }else{
    resp500(w, err)
  }
}

func downloadHandle(w http.ResponseWriter, r *http.Request) {
  if checkPwd(w, r) == 0 {return}
	w.Header().Set("Access-Control-Allow-Origin", "*")
  w.Header().Add("Content-Type", "text/plain")
  if r.Method == "GET" {
    //
	}
	if r.Method == "POST" {
    url := r.FormValue("url")
    filename := r.FormValue("filename")
    downloadpath := filepath.Dir(filename)
    logger.Println(downloadpath)
    err := CreateDir(downloadpath)
    if err == nil {				
      if len(url) > 0 {
        reg, _ := regexp.Compile(`(?i)^data:image/(.+?);base64,(.+)`)
        m := reg.FindStringSubmatch(url)
        if len(m) > 1 { /*** base 64 */
          f := downloadBase64(m[2], filename, m[1])
          io.WriteString(w, f)
        } else {
          logger.Printf("found and download url: %s %s\r\n", url) // %q
          f := downlaodFile(url, filename)
          io.WriteString(w, f)
        }
      }
    }else{
      resp500(w, err)
    }
  }
}

func downloadBase64(source string, filepath string, ext string) string{
  fullpath := filepath // + "." + ext
  b, _ := base64.StdEncoding.DecodeString(source)
  f, _ := os.Create(filepath)
  binary.Write(f, binary.LittleEndian, b)
  return fullpath
}

/* DOWNLAOD FILE FROM URL TO LOCAL */
func downlaodFile(url string, filepath string) string{
  client := &http.Client{}
  request, err := http.NewRequest("GET", url, nil)

  res, err := client.Do(request)
  // res, err := http.Get(url)
  if err != nil {
    logger.Println(err.Error())
		return "" 
  }
 
  f, err := os.Create(filepath)
  // f, err := os.Create(filepath)
  if err != nil {  
		logger.Println(err.Error())
		return "" 
  }
  defer f.Close()
	io.Copy(f, res.Body)
	return filepath
}

func logMessage(message string, printAsWell bool){
  // fmt.Printf / fmt.Println / fmt.Sprint
  logger.Println(message)
  if printAsWell {
    fmt.Println(message)  
  }
}

func startWebServer(addr string, outputJson bool) {
  logMessage(fmt.Sprintf("Create web server on address %s", addr), !outputJson)
  
  if srv != nil {
    // if srv.Addr == addr {
    srv.Shutdown(context.TODO())
    // srv.Close()
    srv = nil
    web_err = nil
    // }
  }
  srv = &http.Server{
    Addr:           addr,
    Handler:        nil,
    ReadTimeout:    10 * time.Second,
    WriteTimeout:   10 * time.Second,
    MaxHeaderBytes: 1 << 20,
  }
  logMessage("Start Server and hold", !outputJson)

  go func() {
    web_err = srv.ListenAndServe() // waiting...
  }()
  
  // defer srv.Shutdown(nil)
  if web_err != nil {
    logMessage(fmt.Sprintf("Listen Error: %s", web_err), !outputJson)
  }
  
  time.Sleep(time.Duration(1) * time.Second)
  web_addr = addr
  if outputJson {
    outputServerInfo()
  }

  // logMessage("Web server started", !outputJson)
}

func outputServerInfo() {
  m := Message{"0", "0", "0", "0", "0", "0"}
  m.Version = version
  m.Serverstate = "ok"
  m.Serveraddr = web_addr
  if web_err != nil {
    m.Serverstate = "fail"
    m.Error = web_err.Error()
  }
  b, _ := json.Marshal(m)
  sendMsgBytes(b)
}

func serverInfoHandle(w http.ResponseWriter, r *http.Request){
  if checkPwd(w, r) == 0 {return}
  w.Header().Add("Content-Type", "text/json")
  m := Message{"0", "0", "0", "0", "0", "0"}
  m.Version = version
  m.Serverstate = "ok"
  m.Serveraddr = web_addr
  if web_err != nil {
    m.Serverstate = "fail"
    m.Error = web_err.Error()
  }
  b, _ := json.Marshal(m)
  io.WriteString(w, string(b))
}

func deleteDirHandle(w http.ResponseWriter, r *http.Request){
  if checkPwd(w, r) == 0 {return}
  w.Header().Add("Content-Type", "text/plain")
  if r.FormValue("path") == "" {
    return
  }
  path := r.FormValue("path")
  is, _ := isDir(path)
  if is {
    os.RemoveAll(path)
  }
}

func isFileHandle(w http.ResponseWriter, r *http.Request){
  if checkPwd(w, r) == 0 {return}
  w.Header().Add("Content-Type", "text/plain")
  path := r.FormValue("path")
  b := "no"
  if IsFile(path) {
    b = "yes"
  }
  // logger.Printf("%s %s\n", path, b)
  io.WriteString(w, b)
}

func checkPwd(w http.ResponseWriter, r *http.Request) (int){
  if web_pwd != "" {
    pwd := r.FormValue("pwd") // from GET or POST or put ...
    // pwd := ""
    // if r.Method == "GET"{
    //   query := r.URL.Query()
    //   pwd = query.Get("pwd")
    // }else{
    //   // POST, GET, PUT and etc (for all requests):
    //   err := r.ParseForm()
    //   if err != nil {
    //     // in case of any error
    //   }else{
    //     pwd = r.Form.Get("pwd")
    //     logger.Println(fmt.Sprintf("get pwd : %s",pwd))
    //   }
    // }
    if pwd != web_pwd {
      resp500(w, errors.New(fmt.Sprintf("WRONG PASSWORD")))
      return 0
    }
  }
  return 1
}

func rootFsHandle(w http.ResponseWriter, r *http.Request){
  params := mux.Vars(r)
  path := params["path"]
  pwd := params["pwd"]
  if web_pwd != "" {
    if pwd != web_pwd {
      resp500(w, errors.New(fmt.Sprintf("WRONG PASSWORD")))
      return
    }
  }
  if runtime.GOOS != "windows" { // darwin, linux, freebsd ...
    path = "/" + path
  }
  // logger.Printf("name %s \n", path)
  http.ServeFile(w, r, path)
}

func dataFsHandle(w http.ResponseWriter, r *http.Request){
  if checkPwd(w, r) == 0 {return}
  rp := r.FormValue("rdf_path")
  f := filepath.Join(rp, "data", strings.TrimPrefix(r.URL.Path, "/data"))
  http.ServeFile(w, r, f)
}

func setRdfPathHandle(w http.ResponseWriter, r *http.Request){
  if checkPwd(w, r) == 0 {return}
  w.Header().Add("Content-Type", "text/plain")
  rdf_path := r.FormValue("path")
  logger.Printf("switch to rdf path %s \r\n", rdf_path)
  // io.WriteString(w, "ok")
}

func fileManagerHandle(w http.ResponseWriter, r *http.Request){
  if checkPwd(w, r) == 0 {return}
  w.Header().Add("Content-Type", "text/plain")
  p := r.FormValue("path")
  var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", p).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", p).Start()
	case "darwin":
		err = exec.Command("open", p).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
  if err != nil {
    io.WriteString(w, err.Error())
    log.Fatal(err)
  }else{
    io.WriteString(w, "ok")
  }
}

func fsCopyHandle(w http.ResponseWriter, r *http.Request){
  if checkPwd(w, r) == 0 {return}
  w.Header().Add("Content-Type", "text/plain")
  src := r.FormValue("src")
  dest := r.FormValue("dest")
  destpath := filepath.Dir(dest)
  err := CreateDir(destpath)
  if err == nil {
    err := copyFsNode(src, dest)
    if err == nil {
      io.WriteString(w, "ok")
    }else{
      resp500(w, err)
    }
  }else{
    resp500(w, err)
  }
}

func fsMoveHandle(w http.ResponseWriter, r *http.Request){
  if checkPwd(w, r) == 0 {return}
  w.Header().Add("Content-Type", "text/plain")
  src := r.FormValue("src")
  dest := r.FormValue("dest")
  err := copyFsNode(src, dest)
  if err == nil {
    err = rmFsNode(src)
  }
  if err != nil {
    io.WriteString(w, err.Error())
  }else{
    io.WriteString(w, "ok")
  }
}

/* ========== MAIN ENTRIES ========== */
func main(){
  // flag.Parse()
  // if *cpuprofile != "" {
  //   f, err := os.Create(*cpuprofile)
  //   if err != nil {
  //     log.Fatal(err)
  //   }
  //   // pprof.NoShutdownHook(*cpuprofile)
  //   // defer f.Close()
  //   pprof.StartCPUProfile(f)
  //   defer pprof.StopCPUProfile()

  //   c := make(chan os.Signal, 2)                    
  //   signal.Notify(c, os.Interrupt, syscall.SIGTERM) // subscribe to system signals
  //   onKill := func(c chan os.Signal) {
  
  //     select {
  //     case <-c:
  //       // defer f.Close()
  //       pprof.StopCPUProfile()
  //       defer os.Exit(0)
  //     }
  //   }
  //   // try to handle os interrupt(signal terminated)
  //   go onKill(c)
  // }  
  
  print(fmt.Sprintf("ScrapBee %s\n", version))
  
  /** log */
	logfile,err:=os.OpenFile("scrapbee_backend.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
  if err!=nil{
    fmt.Printf("%s\r\n",err.Error())
    os.Exit(-1)
  }
  defer logfile.Close()
  logger = log.New(logfile,"",log.Ldate|log.Ltime|log.Lshortfile)
  logger.Println("Start backend")
  
  /** handles by mux */
  rtr := mux.NewRouter()
  rtr.HandleFunc("/file-service/pwd/{pwd:\\w+}/{path:.+}", rootFsHandle).Methods("GET")
  http.Handle("/", rtr)
  
  /** handles by http */
  http.HandleFunc("/isfile/", isFileHandle)
  http.HandleFunc("/deletedir/", deleteDirHandle)
  http.HandleFunc("/filemanager/", fileManagerHandle)
  http.HandleFunc("/download", downloadHandle)
  http.HandleFunc("/savefile", saveFileHandle)
  http.HandleFunc("/savebinfile", saveBinFileHandle)
  http.HandleFunc("/fs/copy", fsCopyHandle)
  http.HandleFunc("/fs/move", fsMoveHandle)
  http.HandleFunc("/serverinfo/", serverInfoHandle)

  arg_start := 1
  /** commmand line args */
  if len(os.Args) > 1 + arg_start && os.Args[1 + arg_start] == "web-server" {
    port := "9900";
    host := "127.0.0.1"
    if len(os.Args) > 2 + arg_start {
      port = os.Args[2 + arg_start]
    }
    if len(os.Args) > 3 + arg_start {
      host = os.Args[3 + arg_start]
    }
    if len(os.Args) > 4 + arg_start{
      web_pwd = os.Args[4 + arg_start]
      fmt.Printf("Password set: %s\r\n", web_pwd)
    }
    addr := fmt.Sprintf("%s:%s", host, port);
    go startWebServer(addr, false)
  }
	// var msg []byte
  var message map[string]string
  var buf []byte = make([]byte, 1024)
  var lenMsg int;
	for {   /** main loop for message interface */
    time.Sleep(time.Duration(1) * time.Second)

    lenMsg = getMsg(buf)
    if lenMsg > 0 {
      // msg = buf[1:lenMsg]
      // logger.Println(fmt.Sprintf("json string: %s", string(msg)))
      unscaped_str, err := strconv.Unquote("\"" + string(buf[1:lenMsg]) + "\"")
      if err != nil {
        logger.Println(fmt.Sprintf("Unquote error: %s", err.Error()))
        continue
      }
      // logger.Println(unscaped_str)
      /**** un-stringify the json string */
      
      if err := json.Unmarshal([]byte(unscaped_str), &message); err != nil {
        logger.Println(fmt.Sprintf("Unmarshal error: %s", err.Error()))
        continue
      }
      // logger.Println(fmt.Sprintf("msgid=%s", message["msgid"]))
      /**** process commands */
      command := message["command"]
      logger.Println(fmt.Sprintf("Message: command = %s", command))
      if command == "web-server" {
        addr := message["addr"]
        if addr == "" {
          port := message["port"]
          addr = fmt.Sprintf("127.0.0.1:%s", port);
        }
        if pwd, ok := message["pwd"]; ok {
          web_pwd = pwd
          if pwd != "" {
            logger.Println(fmt.Sprintf("Password set: %s", web_pwd))
          }
        }        
        go startWebServer(addr, true)
      }
    }
	}
}


func getMsg (b []byte) int{
	inputReader := bufio.NewReader(os.Stdin)	
  // s, err := inputReader.Peek(4)  
	for {
		s, err := inputReader.Peek(4)
    if err != nil{
      // logger.Println(fmt.Sprintf("Error %s", err))
      // b := make([]byte, 0)
      continue
    }
		if s[0] > 0 {
			inputReader.Discard(4)
			// b := make([]byte, s[0])
			_, _ = inputReader.Read(b)
			// return b[1:len(b)-1]
      return int(s[0]) - 1
		}
	}
}

type Message struct {
  Version string
	Rdfloaded string
	Serveraddr string
	Serverstate string
	Downloadjs string
  Error string
}

func sendMsgBytes (arr []byte) {
	var l []byte
	l = []byte{byte((len(arr)>>0)&0xFF), byte((len(arr)>>8)&0xFF), byte((len(arr)>>16)&0xFF), byte((len(arr)>>32)&0xFF)}
	// fmt.Println("s:", []byte(s), "a:", a, " arr:", arr, " len(arr): ", len(arr)>>8, " cap(arr): ", cap(arr), " l: ", l)
	os.Stdout.Write(l);
	os.Stdout.Write(arr);
}

/* copy folder */
func copyFolder(source string, dest string) (err error) {
	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}
	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}
	directory, _ := os.Open(source)
	objects, err := directory.Readdir(-1)
	for _, obj := range objects {
		sourcefilepointer := source + "/" + obj.Name()
		destinationfilepointer := dest + "/" + obj.Name()
		if obj.IsDir() {
			err = copyFolder(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			err = copyFile(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
	return
}

/* copy file */
func copyFile(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourcefile.Close()
	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destfile.Close()
	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
		}
	}
	return
}

/* rm file or folder */
func rmFsNode(src string) (err error){
  is_dir, err := isDir(src)
  if err == nil {
    if(is_dir){
      err = os.RemoveAll(src)
    }else{
      err = os.Remove(src)
    }
  }
  return err
}

/* copy file or folder */
func copyFsNode(src string, dest string) (err error){
  /** always remove dest if already exists */
  if _, err := os.Stat(dest); !os.IsNotExist(err) {
    rmFsNode(dest)
  }
  is_dir, err := isDir(src)
	if err == nil {
    if(is_dir){
      err = copyFolder(src, dest)
    }else{
      err = copyFile(src, dest)
    }
	}
  return err
}
