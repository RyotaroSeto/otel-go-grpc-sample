# otel-go-sample

1. curlでbffにrequestする
2. bffからtodoにrequestする
3. todoからgreetにrequestする
4. greetでrequestを受け取り、todoにresponseを返す
5. todoからbffにresponseを返す
6. bffからcurl実行者にresponseを返す

## 実行
1. localhost:8080/todo

## 流れ
```mermaid

stateDiagram
  User --> Bff: http1.1
  Bff --> Todo: http2.0
  Todo --> Greet: http2.0

```

## 分散トレース
```mermaid

stateDiagram
  User --> Bff: X秒
  Bff --> User: X秒
  Bff --> Todo: X秒
  Todo --> Bff: X秒
  Todo --> Greet: X秒
  Greet --> Todo: X秒

```
