// Supplied diagnostic executable; frozen control is exact 100. Does not print requests, URLs or headers.
package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"
    "time"
)
func main() {
    ds := flag.Bool("ds",false,"POST synthetic control input to configured DS")
    flag.Parse()
    target := "http://127.0.0.1:8080/health"
    method := http.MethodGet
    var body io.Reader
    if *ds {
        base:=os.Getenv("DS_BASE_URL")
        if base=="" { fmt.Println("probe=missing_configuration"); os.Exit(1) }
        target=strings.TrimRight(base,"/")+"/score"
        method=http.MethodPost
        body=bytes.NewBufferString(`{"resume_text":"Go PostgreSQL","vacancy_text":"Go PostgreSQL Docker"}`)
    }
    req,err:=http.NewRequest(method,target,body)
    if err!=nil { fmt.Println("probe=invalid_configuration"); os.Exit(1) }
    if *ds { req.Header.Set("Content-Type","application/json") }
    client:=&http.Client{Timeout:3*time.Second,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}
    resp,err:=client.Do(req)
    if err!=nil { fmt.Println("probe=transport_failed"); os.Exit(1) }
    defer resp.Body.Close()
    if resp.StatusCode!=200 { fmt.Printf("probe=http_failed status=%d\n",resp.StatusCode); os.Exit(1) }
    data,err:=io.ReadAll(io.LimitReader(resp.Body,16385))
    if err!=nil || len(data)>16384 { fmt.Println("probe=body_failed"); os.Exit(1) }
    if *ds {
        var result struct {Score *int `json:"score"`}
        decoder:=json.NewDecoder(bytes.NewReader(data));decoder.DisallowUnknownFields()
        if decoder.Decode(&result)!=nil || result.Score==nil || *result.Score!=100 {
            fmt.Println("probe=unexpected_control_result"); os.Exit(1)
        }
        var extra any
        if decoder.Decode(&extra)!=io.EOF { fmt.Println("probe=trailing_data");os.Exit(1) }
        fmt.Println("probe=ok operation=score expected=100")
    } else { fmt.Println("probe=ok operation=health") }
}
