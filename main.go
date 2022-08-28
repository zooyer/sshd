package main

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	"github.com/zooyer/embed/log"
	"github.com/zooyer/sshd/common/conf"
	gossh "golang.org/x/crypto/ssh"
)

var hashKey = "1F44A914B7894F5BB9A8AA9B53B3FDFB"

// shells 默认shell选择，优先级按顺序排序
var shells = []string{
	"/bin/zsh",
	"/bin/bash",
	"/bin/dash",
	"/bin/ksh",
	"/bin/ash",
	"/bin/tcsh",
	"/bin/csh",
	"/bin/sh",
}

// fileExists 判断文件是否存在
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		return false
	}

	return true
}

// setWinSize 设置终端大小
func setWinSize(f *os.File, w, h int) {
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

// publicKey 验证公钥
func publicKey(ctx ssh.Context, key ssh.PublicKey) bool {
	return false
}

// signer 创建key 来验证 host public
func signer() (gossh.Signer, error) {
	// 如果key文件不存在，则执行ssh-keygen创建
	if _, err := os.Stat(conf.Key); os.IsNotExist(err) {
		// 执行ssh-keygen
		stderr, err := exec.Command("ssh-keygen", "-f", conf.Key, "-t", "rsa", "-N", "").CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("fail to generate private key: %w - %s", err, stderr)
		}
	}

	// 读取文件内容
	data, err := ioutil.ReadFile(conf.Key)
	if err != nil {
		return nil, err
	}

	// 生成ssh.Signer
	return gossh.ParsePrivateKey(data)
}

// hashPassword 对密码进行hash
func hashPassword(password string) string {
	var key = md5.Sum([]byte(hashKey))
	var hash = hmac.New(sha1.New, key[:])

	return hex.EncodeToString(hash.Sum([]byte(password)))
}

// password 密码验证
func password(ctx ssh.Context, password string) bool {
	username := ctx.User()
	if conf.User[username] == "" || password == "" {
		return false
	}

	return conf.User[username] == password || conf.User[username] == hashPassword(password)
}

// handle 核心处理流程
func handle(s ssh.Session) {
	var shell = conf.Shell
	if shell == "" {
		if s := os.Getenv("SHELL"); s != "" {
			shell = s
		}
	}

	// 未找到默认环境变量中的shell，选择一个已存在的shell
	if shell == "" {
		for _, s := range shells {
			if fileExists(s) {
				shell = s
				break
			}
		}
	}

	// 未找到shell，断开连接
	if shell == "" || !fileExists(shell) {
		_, _ = io.WriteString(s, "not found shell.\n")
		return
	}

	cmd := exec.Command(shell)
	cmd.Env = os.Environ()

	ptyReq, winCh, isPty := s.Pty()
	if isPty {
		_, _ = io.WriteString(s, fmt.Sprintln(conf.Banner))
		cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
		for _, env := range conf.Env {
			cmd.Env = append(cmd.Env, env)
		}

		f, err := pty.Start(cmd)
		//f, err := Start(cmd)
		if err != nil {
			panic(err)
		}
		go func() {
			for win := range winCh {
				setWinSize(f, win.Width, win.Height)
			}
		}()

		go io.Copy(f, s) // stdin
		io.Copy(s, f)    // stdout
		if err = cmd.Wait(); err != nil {
			_, _ = io.WriteString(s.Stderr(), err.Error())
			_ = s.Exit(1)
		}
	} else {
		_, _ = io.WriteString(s, "no pty requested.\n")
		_ = s.Exit(1)
	}
}

func main() {
	var err error

	sshd := ssh.Server{
		Addr:            conf.Addr,
		Handler:         handle,
		PasswordHandler: password,
		//PublicKeyHandler: publicKey,
	}

	key, err := signer()
	if err != nil {
		log.ZError("signer error:", err.Error())
		return
	}

	sshd.AddHostKey(key)

	log.ZError("listen error:", sshd.ListenAndServe())
}
