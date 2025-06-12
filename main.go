package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var languages = []string{"go", "python", "php", "java"}

// simple websocket connection handling only text frames
func wsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") != "websocket" {
		http.Error(w, "upgrade required", http.StatusUpgradeRequired)
		return
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	acceptKey := computeAcceptKey(key)

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		return
	}
	defer conn.Close()

	fmt.Fprintf(buf, "HTTP/1.1 101 Switching Protocols\r\n")
	fmt.Fprintf(buf, "Upgrade: websocket\r\n")
	fmt.Fprintf(buf, "Connection: Upgrade\r\n")
	fmt.Fprintf(buf, "Sec-WebSocket-Accept: %s\r\n\r\n", acceptKey)
	buf.Flush()

	for {
		payload, err := readFrame(buf)
		if err != nil {
			if err != io.EOF {
				log.Println("read:", err)
			}
			break
		}
		var req ExecRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			log.Println("bad json:", err)
			continue
		}
		resp := execCode(req)
		data, _ := json.Marshal(resp)
		if err := writeFrame(buf, data); err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func computeAcceptKey(key string) string {
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h[:])
}

func readFrame(r *bufio.ReadWriter) ([]byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	masked := header[1]&0x80 != 0
	length := int(header[1] & 0x7F)
	switch length {
	case 126:
		ext := make([]byte, 2)
		if _, err := io.ReadFull(r, ext); err != nil {
			return nil, err
		}
		length = int(binary.BigEndian.Uint16(ext))
	case 127:
		ext := make([]byte, 8)
		if _, err := io.ReadFull(r, ext); err != nil {
			return nil, err
		}
		length = int(binary.BigEndian.Uint64(ext))
	}
	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(r, maskKey[:]); err != nil {
			return nil, err
		}
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	if masked {
		for i := 0; i < length; i++ {
			payload[i] ^= maskKey[i%4]
		}
	}
	return payload, nil
}

func writeFrame(w *bufio.ReadWriter, payload []byte) error {
	w.WriteByte(0x81) // FIN + text frame
	if len(payload) < 126 {
		w.WriteByte(byte(len(payload)))
	} else if len(payload) < 65536 {
		w.WriteByte(126)
		var buf [2]byte
		binary.BigEndian.PutUint16(buf[:], uint16(len(payload)))
		w.Write(buf[:])
	} else {
		w.WriteByte(127)
		var buf8 [8]byte
		binary.BigEndian.PutUint64(buf8[:], uint64(len(payload)))
		w.Write(buf8[:])
	}
	w.Write(payload)
	return w.Flush()
}

func languagesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(languages)
}

type ExecRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

type ExecResponse struct {
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

func execCode(req ExecRequest) ExecResponse {
	switch req.Language {
	case "go":
		return execGo(req.Code)
	case "python":
		return execCommand("python3", "-c", req.Code)
	case "php":
		return execCommand("php", "-r", req.Code)
	case "java":
		return execJava(req.Code)
	default:
		return ExecResponse{Error: "unsupported language"}
	}
}

func execCommand(name string, args ...string) ExecResponse {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ExecResponse{Output: string(output), Error: err.Error()}
	}
	return ExecResponse{Output: string(output)}
}

func execGo(code string) ExecResponse {
	tmpDir, err := ioutil.TempDir("", "gocode")
	if err != nil {
		return ExecResponse{Error: err.Error()}
	}
	defer os.RemoveAll(tmpDir)

	path := tmpDir + "/main.go"
	if err := ioutil.WriteFile(path, []byte(code), 0644); err != nil {
		return ExecResponse{Error: err.Error()}
	}

	cmd := exec.Command("go", "run", path)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ExecResponse{Output: string(output), Error: err.Error()}
	}
	return ExecResponse{Output: string(output)}
}

func execJava(code string) ExecResponse {
	tmpDir, err := ioutil.TempDir("", "javacode")
	if err != nil {
		return ExecResponse{Error: err.Error()}
	}
	defer os.RemoveAll(tmpDir)

	srcPath := tmpDir + "/Main.java"
	if err := ioutil.WriteFile(srcPath, []byte(code), 0644); err != nil {
		return ExecResponse{Error: err.Error()}
	}

	cmd := exec.Command("javac", "Main.java")
	cmd.Dir = tmpDir
	outCompile, err := cmd.CombinedOutput()
	if err != nil {
		return ExecResponse{Output: string(outCompile), Error: err.Error()}
	}

	cmd = exec.Command("java", "Main")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ExecResponse{Output: string(output), Error: err.Error()}
	}
	return ExecResponse{Output: string(output)}
}

func main() {
	http.HandleFunc("/languages", languagesHandler)
	http.HandleFunc("/ws", wsHandler)
	http.Handle("/", http.FileServer(http.Dir("public")))
	fmt.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
