package template

const DeploymentTemplate = `
# Definition del Service
apiVersion: v1
kind: Service
metadata:
  name: $ROLE
  namespace: $NS
spec:
  selector:
    role: $ROLE
  ports:
    - name: http
      protocol: TCP
      port: $PORT # Puerto en el Service
      targetPort: $PORT # Puerto en el Pod
    - name: rpc
      protocol: TCP
      port: 4200 # Puerto en el Service (RPC)
      targetPort: 4200 # Puerto en el Pod
  type: ClusterIP

---
# Definition del Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $ROLE
  namespace: $NS
spec:
  replicas: $REPLICAS # Número de réplicas
  revisionHistoryLimit: $HISTORY_LIMIT # Limitar de revisiones
  selector:
    matchLabels:
      role: $ROLE # Selector que debe coincidir con los labels de los pods
  template:
    metadata:
      labels:
        role: $ROLE
    spec:
      imagePullSecrets:
        - name: dockerhub-secret
      containers:
        - name: $ROLE
          image: $IMAGE
          imagePullPolicy: Always # Siempre descarga la imagen más reciente
          resources:
            requests:
              cpu: "$CPU_REQUEST" # Recursos mínimos requeridos
              memory: "$MEMORY_REQUEST"
            limits:
              cpu: "$CPU_LIMIT" # Recursos máximos permitidos
              memory: "$MEMORY_LIMIT"
          ports:
            - containerPort: $PORT # Puerto expuesto en el contenedor
          env:
            - name: APP
              value: "$APP"
            - name: PORT
              value: "$PORT"
            - name: VERSION
              value: "$RELEASE"
            - name: PRODUCTION
              value: "$PRODUCTION"
            - name: RPC_HOST
              value: "$ROLE"
            - name: RPC_PORT
              value: "4200"
            - name: TENANT_ID
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: TENANT_ID
            - name: PROJECT_NAME
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: PROJECT_NAME
            - name: RT_URL
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: RT_URL
            - name: WS_USERNAME
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: WS_USERNAME
            - name: WS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: secret-$NS
                  key: WS_PASSWORD
            - name: USER_ADMIN
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: USER_ADMIN
            - name: USER_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: secret-$NS
                  key: USER_PASSWORD
            - name: AUTHORIZATION_METHOD
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: AUTHORIZATION_METHOD
            - name: COMPANY
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: COMPANY
            - name: STAGE
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: STAGE
            - name: WEB
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: WEB
            - name: PATH_URL
              value: "$PATH_URL"
            - name: HOST
              value: "$HOST"
            - name: REQUESTS_LIMIT
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: REQUESTS_LIMIT
            - name: DEBUG
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: DEBUG
            # DB
            - name: DB_DRIVER
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: DB_DRIVER
            - name: DB_HOST
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: DB_HOST
            - name: DB_PORT
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: DB_PORT
            - name: DB_NAME
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: DB_NAME
            - name: DB_USER
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: DB_USER
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: secret-$NS
                  key: DB_PASSWORD            
            # REDIS
            - name: REDIS_HOST
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: REDIS_HOST
            - name: REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: secret-$NS
                  key: REDIS_PASSWORD
            - name: REDIS_DB
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: REDIS_DB
            # NATS
            - name: NATS_HOST
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: NATS_HOST
            - name: NATS_USER
              valueFrom:
                configMapKeyRef:
                  name: config-$NS
                  key: NATS_USER
            - name: NATS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: secret-$NS
                  key: NATS_PASSWORD
            # SECRET
            - name: SECRET
              valueFrom:
                secretKeyRef:
                  name: secret-$NS
                  key: SECRET

  strategy:
    type: RollingUpdate # Actualización gradual (opcional)
    rollingUpdate:
      maxUnavailable: $MAX_PODS_AVAILABLE # Número máximo de pods no disponibles durante la actualización
      maxSurge: $MAX_PODS_SURGE # Número máximo de pods adicionales creados durante la actualización
`
