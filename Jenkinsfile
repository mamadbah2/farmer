pipeline {
    agent any

    environment {
        IMAGE_NAME = 'farmer-bot'
        CONTAINER_NAME = 'farmer-bot'
        PORT = '4040'
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Build Docker Image') {
            steps {
                script {
                    // Build de l'image localement
                    docker.build("${IMAGE_NAME}:${BUILD_NUMBER}")
                    // Tag as latest for convenience
                    sh "docker tag ${IMAGE_NAME}:${BUILD_NUMBER} ${IMAGE_NAME}:latest"
                }
            }
        }

        stage('Deploy') {
            steps {
                script {
                    // Arrêter et supprimer l'ancien conteneur
                    sh "docker stop ${CONTAINER_NAME} || true"
                    sh "docker rm ${CONTAINER_NAME} || true"

                    // Lancer le nouveau conteneur avec les secrets injectés
                    withCredentials([
                        file(credentialsId: 'google-sheets-credentials', variable: 'GOOGLE_CREDS'),
                        string(credentialsId: 'anthropic-api-key', variable: 'ANTHROPIC_KEY'),
                        string(credentialsId: 'whatsapp-token', variable: 'WHATSAPP_TOKEN'),
                        string(credentialsId: 'verify-token', variable: 'VERIFY_TOKEN')
                    ]) {
                        sh """
                            docker run -d \
                            --name ${CONTAINER_NAME} \
                            --restart always \
                            -p ${PORT}:${PORT} \
                            -v \${GOOGLE_CREDS}:/app/credentials.json \
                            -e ANTHROPIC_API_KEY=\${ANTHROPIC_KEY} \
                            -e WHATSAPP_TOKEN=\${WHATSAPP_TOKEN} \
                            -e VERIFY_TOKEN=\${VERIFY_TOKEN} \
                            ${IMAGE_NAME}:latest
                        """
                    }
                }
            }
        }
    }

    post {
        success {
            echo 'Deployment successful!'
        }
        failure {
            echo 'Deployment failed.'
        }
        always {
            cleanWs()
        }
    }
}
