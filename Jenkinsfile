pipeline {
    agent any

    environment {
        // Remplacez ceci par votre nom d'utilisateur Docker Hub
        DOCKER_HUB_USER = 'mamadbah2' 
        IMAGE_NAME = 'farmer-bot'
        CONTAINER_NAME = 'farmer-bot'
        PORT = '4040'
        // Nom complet de l'image : user/repo
        FULL_IMAGE = "${DOCKER_HUB_USER}/${IMAGE_NAME}"
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Build & Push to Docker Hub') {
            steps {
                script {
                    // Utilise les identifiants 'docker-hub-credentials' configurés dans Jenkins
                    docker.withRegistry('', 'docker-hub-credentials') {
                        // Build de l'image
                        def customImage = docker.build("${FULL_IMAGE}:${BUILD_NUMBER}")
                        
                        // Push avec le numéro de build (pour l'historique)
                        customImage.push()
                        
                        // Push avec le tag 'latest' (pour la prod)
                        customImage.push('latest')
                    }
                }
            }
        }

        stage('Deploy') {
            steps {
                script {
                    // Arrêter et supprimer l'ancien conteneur
                    sh "docker stop ${CONTAINER_NAME} || true"
                    sh "docker rm ${CONTAINER_NAME} || true"
                    
                    // Tirer la dernière image depuis Docker Hub
                    sh "docker pull ${FULL_IMAGE}:latest"

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
                            ${FULL_IMAGE}:latest
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
