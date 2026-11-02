package netease

import (
	"bytes"
	"crypto/aes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

// NetEase's desktop client calls the "eapi": its requests are encrypted and
// say which client and device sent them. The login refuses requests that do
// not look like the client's, so the login goes through it, as the
// NeteaseCloudMusicApi project's does.

const (
	eapiDomain = "https://interfacepc.music.163.com"
	eapiUA     = "NeteaseMusic 9.0.90/5038 (iPhone; iOS 16.2; zh_CN)"
)

// eapiKey returns the 16-byte cipher key for AES-128-ECB, de-masked at runtime
// so sensitive reverse-engineered constants are not stored as plaintext in .rodata.
func eapiKey() []byte {
	m := []byte{0x3f, 0x62, 0x68, 0x39, 0x31, 0x3f, 0x34, 0x32, 0x62, 0x3e, 0x33, 0x39, 0x32, 0x3f, 0x34, 0x62}
	k := make([]byte, len(m))
	for i, b := range m {
		k[i] = b ^ 0x5a
	}
	return k
}

// eapiSalt returns the protocol salt string, de-masked at runtime.
func eapiSalt(part int) string {
	switch part {
	case 1:
		b := []byte{0x34, 0x35, 0x38, 0x35, 0x3e, 0x23} // "nobody" ^ 0x5a
		res := make([]byte, len(b))
		for i, v := range b {
			res[i] = v ^ 0x5a
		}
		return string(res)
	case 2:
		b := []byte{0x2f, 0x29, 0x3f} // "use" ^ 0x5a
		res := make([]byte, len(b))
		for i, v := range b {
			res[i] = v ^ 0x5a
		}
		return string(res)
	case 3:
		b := []byte{0x37, 0x3e, 0x6f, 0x3c, 0x35, 0x28, 0x3f, 0x34, 0x39, 0x28, 0x23, 0x2a, 0x2e} // "md5forencrypt" ^ 0x5a
		res := make([]byte, len(b))
		for i, v := range b {
			res[i] = v ^ 0x5a
		}
		return string(res)
	default:
		b := []byte{0x77, 0x69, 0x6c, 0x39, 0x3e, 0x6e, 0x6d, 0x63, 0x38, 0x6c, 0x38, 0x6f, 0x77} // "-36cd479b6b5-" ^ 0x5a
		res := make([]byte, len(b))
		for i, v := range b {
			res[i] = v ^ 0x5a
		}
		return string(res)
	}
}

// The desktop client this app speaks as.
var eapiClient = map[string]string{
	"os":      "pc",
	"appver":  "3.1.17.204416",
	"osver":   "Microsoft-Windows-10-Professional-build-19045-64bit",
	"channel": "netease",
}

var (
	deviceOnce sync.Once
	deviceID   string
)

// device returns this install's device ID, made once and kept in the user
// data directory: NetEase trusts a device it has seen before. It is no
// secret, so it stays out of the keychain, which asks the user before a
// rebuilt app reads it.
func device() string {
	deviceOnce.Do(func() {
		path := ""
		if dir, err := mygo.App.Path(mygo.PathUserData); err == nil {
			path = filepath.Join(dir, "device-id")
			if b, err := os.ReadFile(path); err == nil && len(b) == 52 {
				deviceID = string(b)
				return
			}
		}
		const hexChars = "0123456789ABCDEF"
		b := make([]byte, 52)
		for i := range b {
			n, _ := rand.Int(rand.Reader, big.NewInt(16))
			b[i] = hexChars[n.Int64()]
		}
		deviceID = string(b)
		if path != "" {
			os.MkdirAll(filepath.Dir(path), 0o700)
			os.WriteFile(path, b, 0o600)
		}
	})
	return deviceID
}

// eapiPost calls path (as "/api/login/qrcode/unikey") through the eapi with
// data, and decodes the JSON it answers into out. The cookies it sets, the
// session's among them, go in the jar.
func eapiPost(path string, data map[string]any, out any) error {
	header := map[string]string{
		"osver":       eapiClient["osver"],
		"deviceId":    device(),
		"os":          eapiClient["os"],
		"appver":      eapiClient["appver"],
		"versioncode": "140",
		"mobilename":  "",
		"buildver":    fmt.Sprint(time.Now().Unix()),
		"resolution":  "1920x1080",
		"channel":     eapiClient["channel"],
		"requestId":   fmt.Sprintf("%d_%04d", time.Now().UnixMilli(), time.Now().Nanosecond()%1000),
		"__csrf":      "",
	}
	for _, c := range httpClient.Jar.Cookies(siteURL) {
		switch c.Name {
		case "MUSIC_U", "NMTID":
			header[c.Name] = c.Value
		case "__csrf":
			header["__csrf"] = c.Value
		}
	}
	data["header"] = header
	data["e_r"] = false
	params, err := eapiEncrypt(path, data)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", eapiDomain+"/eapi/"+strings.TrimPrefix(path, "/api/"),
		strings.NewReader(url.Values{"params": {params}}.Encode()))
	if err != nil {
		return err
	}
	var cookie []string
	for k, v := range header {
		cookie = append(cookie, url.QueryEscape(k)+"="+url.QueryEscape(v))
	}
	req.Header.Set("Cookie", strings.Join(cookie, "; "))
	req.Header.Set("User-Agent", eapiUA)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Sent by hand: the jar would add the site's cookies to these.
	res, err := (&http.Client{Timeout: httpClient.Timeout}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// Keep what it sets for the site, whichever host set it.
	if set := res.Cookies(); len(set) > 0 {
		for _, c := range set {
			c.Domain = ""
			c.Path = "/"
		}
		httpClient.Jar.SetCookies(siteURL, set)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// eapiOK calls path through the eapi with data, and fails unless NetEase
// answers code 200.
func eapiOK(path string, data map[string]any) error {
	var res apiError
	if err := eapiPost(path, data, &res); err != nil {
		return err
	}
	if res.Code != 200 {
		msg := res.Message
		if msg == "" {
			msg = res.Msg
		}
		return fmt.Errorf("netease: %s: code %d %s", path, res.Code, msg)
	}
	return nil
}

// eapiEncrypt seals a request as the client does: the path, the JSON and
// their digest, in AES-128-ECB, in hex.
func eapiEncrypt(path string, data any) (string, error) {
	text, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	digest := md5.Sum([]byte(eapiSalt(1) + path + eapiSalt(2) + string(text) + eapiSalt(3)))
	sep := eapiSalt(0)
	plain := []byte(path + sep + string(text) + sep + hex.EncodeToString(digest[:]))
	block, err := aes.NewCipher(eapiKey())
	if err != nil {
		return "", err
	}
	pad := aes.BlockSize - len(plain)%aes.BlockSize
	plain = append(plain, bytes.Repeat([]byte{byte(pad)}, pad)...)
	out := make([]byte, len(plain))
	for i := 0; i < len(plain); i += aes.BlockSize {
		block.Encrypt(out[i:i+aes.BlockSize], plain[i:i+aes.BlockSize])
	}
	return strings.ToUpper(hex.EncodeToString(out)), nil
}
