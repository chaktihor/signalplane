#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/utsname.h>

static int post_json(const char *url, const char *token, const char *json) {
  char command[8192];
  snprintf(command, sizeof(command),
           "curl -s -X POST '%s' -H 'Authorization: Bearer %s' -H 'Content-Type: application/json' -d '%s'",
           url, token, json);
  return system(command);
}

int main(void) {
  const char *base = getenv("SIGNALPLANE_URL");
  const char *token = getenv("SIGNALPLANE_TOKEN");
  if (base == NULL) base = "http://127.0.0.1:4318";
  if (token == NULL) token = "dev-token";

  struct utsname info;
  uname(&info);

  char host_url[512];
  snprintf(host_url, sizeof(host_url), "%s/api/ingest/hosts", base);

  char metric_url[512];
  snprintf(metric_url, sizeof(metric_url), "%s/api/ingest/metrics", base);

  char host_payload[2048];
  snprintf(host_payload, sizeof(host_payload),
           "{\"id\":\"c-host-probe-1\",\"name\":\"c-host-probe-1\",\"environment\":\"production\",\"region\":\"local\",\"status\":\"online\",\"agentVersion\":\"c-probe-dev\",\"tags\":[\"c\",\"host\"],\"metrics\":{\"cpu\":%.2f,\"memory\":%.2f,\"disk\":%.2f}}",
           29.5, 68.0, 44.0);

  char metric_payload[2048];
  snprintf(metric_payload, sizeof(metric_payload),
           "{\"metrics\":[{\"name\":\"host.cpu.usage\",\"value\":29.5,\"unit\":\"percent\",\"type\":\"gauge\",\"resource\":{\"host\":\"c-host-probe-1\",\"environment\":\"production\",\"region\":\"local\"}},{\"name\":\"host.memory.usage\",\"value\":68.0,\"unit\":\"percent\",\"type\":\"gauge\",\"resource\":{\"host\":\"c-host-probe-1\",\"environment\":\"production\",\"region\":\"local\"}}]}");

  printf("host probe kernel=%s machine=%s\n", info.sysname, info.machine);
  post_json(host_url, token, host_payload);
  post_json(metric_url, token, metric_payload);
  printf("\n");
  return 0;
}

