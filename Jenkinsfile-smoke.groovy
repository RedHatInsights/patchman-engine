@Library("github.com/RedHatInsights/insights-pipeline-lib@v3") _

if (env.CHANGE_ID) {
    execSmokeTest (
        ocDeployerBuilderPath: "patchman",
        ocDeployerComponentPath: "patchman",
        ocDeployerServiceSets: "patchman,ingress,inventory,platform-mq,rbac",
        iqePlugins: ["iqe-patchman-plugin"],
        pytestMarker: "patch_smoke",
    )
}
