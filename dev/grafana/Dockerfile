FROM grafana/grafana:12.3.0

USER root
RUN apk add --no-cache python3 py3-yaml
USER grafana

ENV GF_AUTH_ANONYMOUS_ENABLED=true
ENV GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
ENV GF_AUTH_DISABLE_LOGIN_FORM=true

ADD /dev/grafana/extract_dashboard.py /usr/local/bin
ADD /dashboards/grafana/datasources /etc/grafana/provisioning/datasources
ADD /dashboards/grafana/dashboards /etc/grafana/provisioning/dashboards

ADD /dashboards/app-sre/grafana-dashboard-insights-patchman-engine-general.configmap.yaml /etc/grafana

RUN extract_dashboard.py </etc/grafana/grafana-dashboard-insights-patchman-engine-general.configmap.yaml > /etc/grafana/provisioning/dashboards/grafana-dashboard-insights-patchman-engine-general.json
