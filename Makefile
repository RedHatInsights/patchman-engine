# Only temporary, will be replaced by the ocdeployer tool once we finish up the openchift contaienrs


BUILDFILES=$(wildcard openshift/build/*.yml)
DEPLOYFILES=$(wildcard openshift/deploy/*.yml)
DEPLOY_NAMES=spm-engine-listener spm-engine-webserver spm-engine-database

.PHONY: create deploy-local

create: $(DEPLOYFILES) $(BUILDFILES)
	@for file in $(DEPLOYFILES); do oc process -f $${file} | oc apply -f -; done
	@for file in $(BUILDFILES); do oc process -f $${file} | oc apply -f -; done

deploy-local:
	@for deploy in $(DEPLOY_NAMES); do oc start-build $${deploy} --from-dir=.; done