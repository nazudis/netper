package jumper

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/verzth/go-utils/utils"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Request struct {
	r http.Request
	segments map[string]string
	params Params
	files map[string]interface{}
	header http.Header
	Method string
	ClientIP string
	ClientPort string
}

func PlugRequest(r *http.Request, w http.ResponseWriter) *Request {
	req := &Request{
		r:          *r,
		segments: mux.Vars(r),
		params: Params{},
		files: map[string]interface{}{},
		header: r.Header,
		Method: r.Method,
		ClientIP: getHost(r),
		ClientPort: getPort(r),
	}

	// PARSE QUERY STRING PARAMETERS
	for k, v := range r.URL.Query() {
		req.params[k] = scan(v)
	}

	switch r.Method {
	case http.MethodGet,http.MethodPut,http.MethodPost,http.MethodDelete,http.MethodPatch:{
		contentType := req.header.Get("Content-Type")
		if strings.Contains(contentType, "multipart/form-data") {
			contentType = "multipart/form-data"
		}
		switch contentType {
		case "multipart/form-data":{
			if r.Method == http.MethodGet {
				return req
			}
			err := r.ParseMultipartForm(32 << 10)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return req
			}
			for k, v := range r.MultipartForm.Value {
				req.params[k] = scan(v)
			}
			for k, v := range r.MultipartForm.File {
				req.files[k] = scanFiles(v)
			}
			break
		}
		case "application/x-www-form-urlencoded":{
			if r.Method == http.MethodGet {
				return req
			}
			err := r.ParseForm()
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return req
			}
			for k, v := range r.PostForm {
				req.params[k] = scan(v)
			}
			break
		}
		case "application/json":{
			dec := json.NewDecoder(r.Body)

			err := dec.Decode(&req.params)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return req
			}
			break
		}
		}
		break
	}
	}
	return req
}

func scan(values []string) interface{} {
	if len(values) == 1 {
		return identify(values[0])
	}else if len(values) > 1 {
		list := []interface{}{}
		for k,vs := range values {
			list[k] = identify(vs)
		}
		return list
	} else {
		return nil
	}
}

func identify(value string) interface{} {
	var arr []interface{}
	var mp map[string]interface{}
	errArr := json.Unmarshal([]byte(value), &arr)
	errMp := json.Unmarshal([]byte(value), &mp)
	if errArr == nil {
		return arr
	} else if errMp == nil {
		return mp
	} else {
		return value
	}
}

func scanFiles(values []*multipart.FileHeader) interface{} {
	if len(values) == 1 {
		return values[0]
	}else if len(values) > 1 {
		return values
	} else {
		return nil
	}
}


func (r *Request) GetHost() string {
	return r.r.URL.Hostname()
}

func (r *Request) GetPort() string {
	return r.r.URL.Port()
}

func (r *Request) GetScheme() string {
	return r.r.URL.Scheme
}

func (r *Request) GetOpaque() string {
	return r.r.URL.Opaque
}

func (r *Request) GetPath() string {
	return r.r.URL.Path
}

func (r *Request) GetRawPath() string {
	return r.r.URL.RawPath
}

func (r *Request) GetRawQuery() string {
	return r.r.URL.RawQuery
}

func (r *Request) GetFragment() string {
	return r.r.URL.Fragment
}

func (r *Request) HasUser() bool {
	_,_,ok := r.r.BasicAuth()
	return ok
}

func (r *Request) GetUsername() string {
	user,_,ok := r.r.BasicAuth()
	if ok {
		return user
	}
	return ""
}

func (r *Request) GetPassword() string {
	_,pass,ok := r.r.BasicAuth()
	if ok {
		return pass
	}
	return ""
}

func (r *Request) GetUrl() string {
	return r.r.URL.Scheme+"://"+r.r.URL.Host+r.r.URL.EscapedPath()
}

func (r *Request) GetFullUrl() string {
	return r.r.URL.String()
}

func (r *Request) Header(key string) string {
	return r.header.Get(key)
}

func (r *Request) GetAll() map[string] interface{} {
	return r.params
}

func (r *Request) Get(key string) string {
	val := reflect.ValueOf(r.params[key])
	if r.params[key] != nil || (val.IsValid() && val.Kind() == reflect.Slice && val.Len() > 0){
		return fmt.Sprintf("%v", r.params[key])
	}
	return ""
}

func (r *Request) Append(key string, val string) {
	r.params[key] = val
}

func (r *Request) GetSegment(key string) string {
	return r.segments[key]
}

func (r *Request) GetSegmentUint64(key string) uint64 {
	if r.segments[key] != "" {
		i64, _ := strconv.ParseUint(r.segments[key], 10, 32)
		return i64
	}
	return 0
}

func (r *Request) GetSegmentUint32(key string) uint32 {
	return uint32(r.GetSegmentUint64(key))
}

func (r *Request) GetSegmentUint(key string) uint {
	return uint(r.GetSegmentUint64(key))
}

func (r *Request) GetSegmentInt64(key string) int64 {
	if r.segments[key] != "" {
		i64, _ := strconv.ParseInt(r.segments[key], 10, 32)
		return i64
	}
	return 0
}

func (r *Request) GetSegmentInt32(key string) int32 {
	return int32(r.GetSegmentInt64(key))
}

func (r *Request) GetSegmentInt(key string) int {
	return int(r.GetSegmentInt64(key))
}

func (r *Request) GetFile(key string) (*File, error) {
	if r.files[key] != nil {
		_, ok := r.files[key].(*multipart.FileHeader)
		if ok {
			f, err := r.files[key].(*multipart.FileHeader).Open()
			return &File{
				f:  f,
				fh: r.files[key].(*multipart.FileHeader),
			}, err
		} else {
			return nil, errors.New("invalid file, maybe files instead")
		}
	}
	return nil, errors.New("no such file")
}

func (r *Request) GetFiles(key string) ([]*File, error) {
	if r.files[key] != nil {
		var files []*File
		vs, ok := r.files[key].([]*multipart.FileHeader)
		if ok {
			for _, v := range vs{
				f, err := v.Open()
				if err != nil {
					return nil, errors.New("files error")
				}
				files = append(files, &File{
					f:  f,
					fh: v,
				})
			}
			return files, nil
		} else {
			return nil, errors.New("invalid files, maybe file instead")
		}
	}
	return nil, errors.New("no such file")
}

func (r *Request) GetUint64(key string) uint64 {
	if r.params[key] != nil {
		switch r.params[key].(type) {
		case float64: return uint64(r.params[key].(float64))
		case int: return uint64(r.params[key].(int))
		case string:
			i64, _ := strconv.ParseUint(r.params[key].(string), 10, 32)
			return i64
		case bool: {
			if r.params[key].(bool) {
				return 1
			}else{
				return 0
			}
		}
		}
	}
	return 0
}

func (r *Request) GetUint32(key string) uint32 {
	return uint32(r.GetUint64(key))
}

func (r *Request) GetUint(key string) uint {
	return uint(r.GetUint64(key))
}

func (r *Request) GetInt64(key string) int64 {
	if r.params[key] != nil {
		switch r.params[key].(type) {
		case float64: return int64(r.params[key].(float64))
		case int: return int64(r.params[key].(int))
		case string:
			i64, _ := strconv.ParseInt(r.params[key].(string), 10, 32)
			return i64
		case bool: {
			if r.params[key].(bool) {
				return 1
			}else{
				return 0
			}
		}
		}
	}
	return 0
}

func (r *Request) GetInt32(key string) int32 {
	return int32(r.GetInt64(key))
}

func (r *Request) GetInt(key string) int {
	return int(r.GetInt64(key))
}

func (r *Request) GetFloat64(key string) float64 {
	if r.params[key] != nil {
		switch r.params[key].(type) {
		case float64: return r.params[key].(float64)
		case int: return float64(r.params[key].(int))
		case string:
			i64, _ := strconv.ParseFloat(r.params[key].(string), 10)
			return i64
		case bool: {
			if r.params[key].(bool) {
				return 1
			}else{
				return 0
			}
		}
		}
	}
	return 0
}

func (r *Request) GetFloat(key string) float32 {
	return float32(r.GetFloat64(key))
}

func (r *Request) GetBool(key string) bool {
	if r.params[key] != nil {
		switch r.params[key].(type) {
		case float64: return r.params[key].(float64) > 0
		case int: return float64(r.params[key].(int)) > 0
		case string:
			i64, _ := strconv.ParseFloat(r.params[key].(string), 10)
			return i64 > 0
		case bool: return r.params[key].(bool)
		}
	}
	return false
}

func (r *Request) GetTime(key string) (*time.Time,error) {
	if r.params[key] != nil {
		t, err := time.Parse(time.RFC3339,r.params[key].(string))
		if err != nil {
			return nil, errors.New("use RFC3339 format string for datetime")
		}
		return &t, nil
	} else {
		return nil, errors.New("no time specified")
	}
}

func (r *Request) GetTimeNE(key string) *time.Time {
	t, _ := r.GetTime(key)
	return t
}

func (r *Request) GetArray(key string) []interface{} {
	if r.params[key] != nil {
		if v, ok := r.params[key].([]interface{}); ok {
			return v
		}
	}
	return nil
}

func (r *Request) GetArrayUniquify(key string) []interface{} {
	if r.params[key] != nil {
		if v, ok := r.params[key].([]interface{}); ok {
			utils.Uniquify(&v)
			return v
		}
	}
	return nil
}

func (r *Request) GetMap(key string) map[string]interface{} {
	if r.params[key] != nil {
		if v, ok := r.params[key].(map[string]interface{}); ok {
			return v
		}
	}
	return nil
}

func (r *Request) GetJSON(key string) JSON {
	jsonObj, err := json.Marshal(r.params[key])
	if err != nil {
		return nil
	}else{
		return jsonObj
	}
}

func (r *Request) GetStruct(obj interface{}) error {
	decoder := json.NewDecoder(r.r.Body)
	return decoder.Decode(&obj)
}

func (r *Request) has(key string) bool {
	if _, found := r.params[key]; !found {
		return false
	}
	return true
}

func (r *Request) Has(keys... string) (found bool) {
	found = true
	for _, key := range keys {
		found = found && r.has(key)
	}
	return
}

func (r *Request) Filled(keys... string) (found bool) {
	found = true
	for _, key := range keys {
		found = found && r.has(key)
		val := reflect.ValueOf(r.params[key])
		if val.IsValid() {
			switch val.Kind() {
			case reflect.String: found = found && strings.TrimSpace(r.Get(key)) != ""
			case reflect.Slice: found = found && val.Len() > 0
			}
		}else{
			found = false
		}
	}
	return
}

func (r *Request) hasHeader(key string) bool {
	if _, found := r.header[strings.Title(key)]; !found {
		return false
	}
	return true
}

func (r *Request) HasHeader(keys... string) (found bool) {
	found = true
	for _, key := range keys {
		found = found && r.hasHeader(key)
	}
	return
}

func (r *Request) HeaderFilled(keys... string) (found bool) {
	found = true
	for _, key := range keys {
		found = found && r.hasHeader(key) && r.Header(key) != ""
	}
	return
}

func (r *Request) HasFile(keys... string) (found bool) {
	found = true
	for _, key := range keys {
		found = found && r.files[key] != nil
	}
	return
}