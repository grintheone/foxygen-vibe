export const routePaths = {
  dashboard: "/dashboard",
  profile: "/profile",
  signIn: "/",
  clientPattern: "/clients/:clientId",
  devicePattern: "/devices/:deviceId",
  ticketPattern: "/tickets/:ticketId",
  clientById(clientId) {
    return `/clients/${clientId}`;
  },
  deviceById(deviceId) {
    return `/devices/${deviceId}`;
  },
  ticketById(ticketId) {
    return `/tickets/${ticketId}`;
  },
};
