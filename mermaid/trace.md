```mermaid

stateDiagram
  User --> Bff
  Bff --> Todo
  Todo --> Greet

```


```mermaid

stateDiagram
  User --> Bff: X秒
  Bff --> User: X秒
  Bff --> Todo: X秒
  Todo --> Bff: X秒
  Todo --> Greet: X秒
  Greet --> Todo: X秒

```
