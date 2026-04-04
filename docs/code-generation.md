# Code Generation

restless can generate equivalent code for any request in 8 languages. In the TUI, press `yg` followed by a language key.

## Languages

### Python (`ygp`)
```python
import requests

response = requests.post(
    'https://api.example.com/users',
    headers={
        'Content-Type': 'application/json',
        'Authorization': 'Bearer token123',
    },
    json={"name":"Alice"},
)

print(response.status_code)
print(response.text)
```

### JavaScript (`ygj`)
```javascript
const response = await fetch("https://api.example.com/users", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer token123",
  },
  body: JSON.stringify({"name":"Alice"}),
});

const data = await response.text();
console.log(response.status, data);
```

### Go (`ygg`)
```go
req, err := http.NewRequest("POST", "https://api.example.com/users", strings.NewReader(`{"name":"Alice"}`))
if err != nil {
    log.Fatal(err)
}
req.Header.Set("Content-Type", "application/json")

resp, err := http.DefaultClient.Do(req)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

body, _ := io.ReadAll(resp.Body)
fmt.Println(resp.StatusCode, string(body))
```

### Java (`ygv`)
```java
HttpClient client = HttpClient.newHttpClient();
HttpRequest request = HttpRequest.newBuilder()
    .uri(URI.create("https://api.example.com/users"))
    .method("POST", HttpRequest.BodyPublishers.ofString("{\"name\":\"Alice\"}"))
    .header("Content-Type", "application/json")
    .build();

HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
System.out.println(response.statusCode() + " " + response.body());
```

### Ruby (`ygr`)
```ruby
require 'net/http'
require 'uri'

uri = URI('https://api.example.com/users')
http = Net::HTTP.new(uri.host, uri.port)
http.use_ssl = uri.scheme == 'https'

request = Net::HTTP::Post.new(uri)
request['Content-Type'] = 'application/json'
request.body = '{"name":"Alice"}'

response = http.request(request)
puts "#{response.code} #{response.body}"
```

### HTTPie (`ygh`)
```bash
http POST https://api.example.com/users \
  Content-Type:'application/json' \
  name='Alice'
```

### curl (`ygc`)
```bash
curl -X POST -H 'Content-Type: application/json' --data-raw '{"name":"Alice"}' -L 'https://api.example.com/users'
```

### PowerShell (`ygw`)
```powershell
$headers = @{
    "Content-Type" = "application/json"
}
$body = '{"name":"Alice"}'

$response = Invoke-RestMethod -Uri "https://api.example.com/users" -Method POST -Headers $headers -Body $body
$response
```
