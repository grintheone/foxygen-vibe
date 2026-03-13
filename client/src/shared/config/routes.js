export const routePaths = {
  dashboard: "/dashboard",
  profile: "/profile",
  memberProfilePattern: "/profile/:userId",
  profileArchivePattern: "/profile/:userId/archive",
  signIn: "/",
  clientPattern: "/clients/:clientId",
  clientArchivePattern: "/clients/:clientId/archive",
  clientContactsPattern: "/clients/:clientId/contacts",
  clientAgreementsPattern: "/clients/:clientId/agreements",
  devicePattern: "/devices/:deviceId",
  deviceArchivePattern: "/devices/:deviceId/archive",
  ticketPattern: "/tickets/:ticketId",
  clientById(clientId) {
    return `/clients/${clientId}`;
  },
  clientArchiveById(clientId) {
    return `/clients/${clientId}/archive`;
  },
  clientContactsById(clientId) {
    return `/clients/${clientId}/contacts`;
  },
  clientAgreementsById(clientId) {
    return `/clients/${clientId}/agreements`;
  },
  deviceById(deviceId) {
    return `/devices/${deviceId}`;
  },
  deviceArchiveById(deviceId) {
    return `/devices/${deviceId}/archive`;
  },
  ticketById(ticketId) {
    return `/tickets/${ticketId}`;
  },
  profileById(userId) {
    return `/profile/${userId}`;
  },
  profileArchiveById(userId) {
    return `/profile/${userId}/archive`;
  },
};
