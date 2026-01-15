# {{.ServiceName}} RPC Client Configuration
serviceName: "{{.ServiceLower}}-rpc"

# Service discovery configuration
discovery:
  type: "consul"  # Options: "consul", "etcd", "direct"
  consul:
    address: "localhost:8500"

# Direct endpoints (used when discovery.type is "direct" or "none")
# endpoints:
#   - "localhost:8888"

# Timeout configuration
timeout:
  rpc: 3s
  connect: 1s

# Retry configuration
retry:
  enabled: true
  maxRetries: 2
  delay: 100ms
  maxDelay: 1s

# Circuit breaker configuration
circuitBreaker:
  enabled: true
  errorRate: 0.5
  minSamples: 200

# Load balancer strategy
loadBalancer: "roundrobin"  # Options: "roundrobin", "random", "consistenthash"

# Connection pool settings
maxIdlePerAddress: 10
maxIdleGlobal: 100
maxIdleTimeout: 30s