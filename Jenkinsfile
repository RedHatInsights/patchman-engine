// Jenkinsfile for gh-pr-and-build template
// See: https://www.jenkins.io/doc/book/pipeline/jenkinsfile/

def secrets = [
    // params.VAULT_PATH_SVC_ACCOUNT_EPHEMERAL
    [path: 'insights-cicd/ephemeral-bot-svc-account', engineVersion: 1, secretValues: [
        [envVar: 'OC_LOGIN_TOKEN_DEV', vaultKey: 'oc-login-token-dev'],
        [envVar: 'OC_LOGIN_SERVER_DEV', vaultKey: 'oc-login-server-dev']]],
    // params.VAULT_PATH_QUAY_PUSH
    [path: 'app-sre/quay/app-sre-push', engineVersion: 1, secretValues: [
        [envVar: 'QUAY_USER', vaultKey: 'user'],
        [envVar: 'QUAY_TOKEN', vaultKey: 'token']]],
    // params.VAULT_PATH_RHR_PULL
    [path: 'insights-cicd/rh-registry-pull', engineVersion: 1, secretValues: [
        [envVar: 'RH_REGISTRY_USER', vaultKey: 'user'],
        [envVar: 'RH_REGISTRY_TOKEN', vaultKey: 'token']]]
]

//                             params.VAULT_ADDRESS,                            params.VAULT_CREDS_ID
def configuration = [vaultUrl: "https://vault.devshift.net", vaultCredentialId: 'vault-creds', engineVersion: 1]

pipeline {
    // Agent configuration - defines where the pipeline runs
    agent {
        node {
            // Use spot instances for cost efficiency
            label 'rhel8-spot'
        }
    }

    // Pipeline options
    options {
        // Add timestamps to console output
        timestamps()
    }

    stages {
        // Stage 1: PR Check - runs for pull requests only
        stage('PR Check') {
            when {
                // Only execute when building a pull request
                // Environment variables available: CHANGE_ID, CHANGE_AUTHOR, CHANGE_TARGET, etc.
                changeRequest()
            }
            steps {
                wrap([$class: 'VaultBuildWrapper',
                    vaultSecrets: [
                        [
                            configuration: configuration,
                            secretValues: secrets
                        ]
                    ]
                ]) {
                // Run PR validation script
                sh './pr_check.sh'
                }
            }
        }

        // Stage 2: Build - runs for main branch only
        stage('Build') {
            when {
                // Only execute when building the main branch
                branch 'main'
            }
            steps {
                // VaultBuildWrapper injects secrets as environment variables
                // Secrets are ONLY available in this stage, not in PR Check for security
                wrap([$class: 'VaultBuildWrapper',
                    vaultSecrets: [
                        [
                            configuration: configuration,
                            secretValues: secrets
                        ]
                    ]
                ]) {
                    // Run build/deploy script with access to secrets
                    sh './build_deploy.sh'
                }
            }
        }
    }

    // Post-build actions
    post {
        always {
            // Clean workspace after every build to save disk space
            cleanWs()
        }
    }
}
