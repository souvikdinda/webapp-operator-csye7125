def webappVersion
def webappDBVersion
pipeline {
    agent any 
    
    environment {
        GCLOUD_PROJECT = 'csye7125-gke-f8c10e6f'
        GCLOUD_CLUSTER = 'my-gke-cluster'
        GCLOUD_REGION = 'us-east1'
    }
    
    stages {

        stage('Checking out the code') {
            steps{
                script {
                    checkout([
                        $class: 'GitSCM',
                        branches: [[name: 'main']],
                        userRemoteConfigs: [[credentialsId: 'github_token', url: 'https://github.com/csye7125-fall2023-group07/webapp-operator.git']]
                    ])
                }
            }
        }

        stage('Semantic-Release') {
            steps {
                script {
                    withCredentials([usernamePassword(credentialsId: 'github_token', usernameVariable: 'GH_USERNAME', passwordVariable: 'GH_TOKEN')]) {
                        env.GIT_LOCAL_BRANCH = 'main'
                        sh "mv release.config.js release.config.cjs"
                        sh "npm install @semantic-release/commit-analyzer@8.0.1"
                        sh "npm install @semantic-release/git@9.0.1"
                        sh "npx semantic-release"
                    }
                }
            }
        }

        stage('Build Docker Image') {
            steps {
                script {
                    releaseTag = sh(returnStdout: true, script: 'git describe --tags --abbrev=0').trim()
                    echo "Release tag is ${releaseTag}"
                    sh "make docker-build"
                    sh "docker tag quay.io/csye-7125/webapp-operator:latest quay.io/csye-7125/webapp-operator:${releaseTag}"
                    sh "make docker-push"
                    sh "docker push quay.io/csye-7125/webapp-operator:${releaseTag}"
                }
            }
        }

        stage('Authenticating with GKE') {
            steps {
                script {
                    withCredentials([file(credentialsId: 'gke_service_account_key', variable: 'GCLOUD_SERVICE_KEY')]) {
                        sh "gcloud auth activate-service-account --key-file=${GCLOUD_SERVICE_KEY}"
                    }

                    sh "gcloud config set project ${GCLOUD_PROJECT}"
                    sh "gcloud container clusters get-credentials my-gke-cluster --region ${GCLOUD_REGION}"

                    echo "Authenticated with GKE..."

                }
            }
        }

        stage('Deploying to GKE') {
            steps {
                sh 'make deploy'
            }
        }

        stage('Revoke GKE access') {
            steps {
                sh 'gcloud auth revoke --all'
            }
            
        }
    }
}