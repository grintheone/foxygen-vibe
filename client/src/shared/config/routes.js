export const routePaths = {
  dashboard: "/dashboard",
  profile: "/profile",
  signIn: "/",
  clientPattern: "/clients/:clientId",
  clientArchivePattern: "/clients/:clientId/archive",
  clientContactsPattern: "/clients/:clientId/contacts",
  clientAgreementsPattern: "/clients/:clientId/agreements",
  devicePattern: "/devices/:deviceId",
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
  ticketById(ticketId) {
    return `/tickets/${ticketId}`;
  },
};
