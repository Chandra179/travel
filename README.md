# Online Travel Agency

ota: flights

## Project Structure

```
├── cmd/                               # Runnable applications / entrypoints
│   ├── travel/                        
│   │   └── main.go
├── pkg/                               # Reusable library packages
│   ├── cache/                         # Cache interfaces, Redis helpers, wrappers
│   ├── logger/                        # Zerolog wrapper & helpers
├── api/
├── cfg/                               # Centralized config files (YAML, JSON, HCL)
│   ├── app.yaml
│   └── redis.yaml

```

## Key Points

1. **`cmd/` folder**  
   - Each subdirectory represents a **separate runnable service or example**.  
   - Demonstrates **service configuration and execution**.

2. **`pkg/` folder**  
   - Contains **reusable packages** for core functionality.  
   - Standard Go convention for libraries.  

3. **Usage Examples**  
   - Each service uses reusable logic from `pkg/`.