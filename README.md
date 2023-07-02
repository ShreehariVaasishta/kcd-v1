# kcd
A simple AWS codebuild inspired CI/CD tool on Kubernetes.


## Instructions
There are two parts to this application. 

### 1. client-sdk
A Golang app for creating a pod and accepting the configs.

### 2. runner
A Golang container that runs in the pod created by the client-sdk
