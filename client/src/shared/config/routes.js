export const routePaths = {
  dashboard: "/dashboard",
  profile: "/profile",
  signIn: "/",
  ticketPattern: "/tickets/:ticketId",
  ticketById(ticketId) {
    return `/tickets/${ticketId}`;
  },
};
