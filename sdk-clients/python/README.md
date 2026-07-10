# TelemetryHealth Python SDK Client

A lightweight Python client for status polling and querying service health scores (PRD §7, §13).

## Usage

```python
from telemetryhealth import Client

client = Client(endpoint="http://localhost:8080", tenant_id="your-tenant-uuid")
score = client.get_health_score(service="checkout-service", env="production")
print(f"Current Health Score: {score}")
```
