import ticketAssignedIcon from "../../../assets/icons/ticket-assigned.svg";
import ticketCanceledIcon from "../../../assets/icons/ticket-canceled.svg";
import ticketClosedIcon from "../../../assets/icons/ticket-closed.svg";
import ticketCreatedIcon from "../../../assets/icons/ticket-created.svg";
import ticketDoneIcon from "../../../assets/icons/ticket-done.svg";
import ticketInWorkIcon from "../../../assets/icons/ticket-inwork.svg";

export const statusIconByType = {
    assigned: ticketAssignedIcon,
    canceled: ticketCanceledIcon,
    cancelled: ticketCanceledIcon,
    closed: ticketClosedIcon,
    created: ticketCreatedIcon,
    inWork: ticketInWorkIcon,
    worksDone: ticketDoneIcon,
};

export const FALLBACK_STATUS_ICON = ticketAssignedIcon;
