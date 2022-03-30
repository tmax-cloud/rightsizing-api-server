package main

import (
 "time"

 rotatelogs "github.com/lestrrat-go/file-rotatelogs"
 log "github.com/sirupsen/logrus"
)

func init() {
 path := "./log/go.log"
 /*Log rotation correlation function
 `Withlinkname 'establishes a soft connection for the latest logs
 `Withrotationtime 'sets the time of log splitting, and how often to split
 Only one of withmaxage and withrotationcount can be set
  `Withmaxage 'sets the maximum save time before cleaning the file
  `Withrotationcount 'sets the maximum number of files to be saved before cleaning
 */
 //The following configuration logs rotate a new file every 1 minute, keep the log files of the last 3 minutes, and automatically clean up the surplus.
 writer, _ := rotatelogs.New(
 path+".%Y%m%d%H%M",
 rotatelogs.WithLinkName(path),
 rotatelogs.WithMaxAge(time.Duration(180)*time.Second),
 rotatelogs.WithRotationTime(time.Duration(60)*time.Second),
 )
 log.SetOutput(writer)
 //log.SetFormatter(&log.JSONFormatter{})
}

func main() {
 for {
 log.Info("hello, world!")
 time.Sleep(time.Duration(2) * time.Second)
 }
}
