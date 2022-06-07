# thermostat

Run controller:
```
make install 
make deploy IMG=ghcr.io/pbochynski/thermostat:latest
```

Deploy resources:
```
kubectl apply -f ./config/samples
```