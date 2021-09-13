#include <stdint.h>
#include <librist/librist.h>

extern int goAuthHandlerOnConnect(void *, const char *, uint16_t, const char *, uint16_t, struct rist_peer *);
extern int goAuthHandlerOnDisconnect(void *, struct rist_peer *);
extern void goConnectionStatusHandlerOnConnectionStatus(void *, struct rist_peer *, enum rist_connection_status);
extern int goOobHandlerOnReceiveOob(void *, const struct rist_oob_block *);
extern int goStatsHandlerOnReceiveStats(void *, const struct rist_stats *);
extern int goReceiverDataHandlerOnData(void *, struct rist_data_block *);

int cb_auth_connect(void *arg, const char *connecting_ip, uint16_t connecting_port, const char *local_ip, uint16_t local_port, struct rist_peer *peer)
{
  return goAuthHandlerOnConnect(arg, connecting_ip, connecting_port, local_ip, local_port, peer);
}

int cb_auth_disconnect(void *arg, struct rist_peer *peer)
{
  return goAuthHandlerOnDisconnect(arg, peer);
}

void cb_connection_status(void *arg, struct rist_peer *peer, enum rist_connection_status peer_connection_status)
{
  goConnectionStatusHandlerOnConnectionStatus(arg, peer, peer_connection_status);
}

int cb_recv_oob(void *arg, const struct rist_oob_block *oob_block)
{
  return goOobHandlerOnReceiveOob(arg, oob_block);
}

int cb_stats(void *arg, const struct rist_stats *stats_container)
{
  return goStatsHandlerOnReceiveStats(arg, stats_container);
}

int cb_recv(void *arg, struct rist_data_block *b)
{
  return goReceiverDataHandlerOnData(arg, b);
}