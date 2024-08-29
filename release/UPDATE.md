# Automatic updates (patches and security updates)

Run periodically (e.g. once per day):

```
kubectl apply -f https://<some-asset-host>/crossbind/update-v1.0.yaml
```

# Manual updates (new minor or major version)

1. Update `local-vars` if necessary (e.g. a new setting was added in the release)
2. Apply new `install` bundle (TBD)
3. Reinstall configuration, using new configuration bundle:
4. Change update automation to use the new minor version:
```
kubectl apply -f https://<some-asset-host>/crossbind/update-v1.1.yaml
```
