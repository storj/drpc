pipeline {
    agent {
        docker {
            label 'main'
            image docker.build("storj-ci", "--pull git://github.com/storj/ci.git#main").id
            args '-u root:root --cap-add SYS_PTRACE -v "/tmp/gomod":/go/pkg/mod'
        }
    }

    options {
          timeout(time: 26, unit: 'MINUTES')
    }

    environment {
        GOTRACEBACK = 'all'
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Test') {
            steps {
                sh 'make test'
            }
        }

        stage('Lint') {
            steps {
                sh 'make lint'
            }
        }
    }

    post {
        always {
            sh "chmod -R 777 ." // ensure Jenkins agent can delete the working directory
            deleteDir()
        }
    }
}
