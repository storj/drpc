pipeline {
    agent {
        dockerfile {
            filename 'Dockerfile.jenkins'
            args '-u root:root --cap-add SYS_PTRACE -v "/tmp/gomod":/go/pkg/mod'
            label 'main'
        }
    }
    stages {
        stage('Download') {
            steps {
                checkout scm
                sh './scripts/download.sh'
            }
        }
        stage('Test') {
            steps {
                sh './scripts/test.sh'
            }
        }
        stage('Lint') {
            steps {
                sh './scripts/lint.sh'
            }
        }
    }
}
