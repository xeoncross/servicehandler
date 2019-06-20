
# Service Wrapper

An HTTP mux/router that takes a service and generates the HTTP REST endpoints (with validation) for that service.

The issue with this approach is that services probably shouldn't handle things like access-control or notifications to external services (submitting to a message queue or sending an uptime metric). By having this generate everything, we are losing control over each unique endpoint causing the service we provide to need to handle stuff outside it's domain/scope.


- Ingest service provided
- loop over methods
- for each method
  - create URL endpoint
  - get param types
- on each request, match endpoint (or 404)
- then create param type instances (for absorbing JSON)
- populate with JSON
- validate
- Call method and look for error
