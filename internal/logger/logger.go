package logger

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)


type Level string

const (
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
	LevelDebug Level = "debug"
)

type Logger struct{
  mu sync.Mutex
  output io.Writer
  
}

func New() *Logger{
	return &Logger{
		output: os.Stdout,
	}
}

func NewWithWriter(w io.Writer) *Logger {
	return &Logger{
		output: w,
	}
}

func (l *Logger) Log(level Level, msg string, fields map[string]interface{}) {
   entry:=make(map[string]interface{})
   entry["level"] = string(level)
   entry["msg"] = msg
   entry["time"] = time.Now()

   for k,v:= range fields {
	 entry[k]=v
   }

   l.mu.Lock()
   defer l.mu.Unlock()

   data,err:=json.Marshal(entry)
   if err!=nil{
	return
   }
   l.output.Write(data)
   l.output.Write([]byte("\n"))
}

func (l *Logger) Info(msg string,fields map[string]interface{}) {
	l.Log(LevelInfo,msg,fields)

}

func (l *Logger) Warn(msg string,fields map[string]interface{}) {
	l.Log(LevelWarn,msg,fields)

}

func (l *Logger) Error(msg string,fields map[string]interface{}) {
	l.Log(LevelError,msg,fields)

}

func (l *Logger) Debug(msg string,fields map[string]interface{}) {
	l.Log(LevelDebug,msg,fields)

}	

var Global =New()


   	

