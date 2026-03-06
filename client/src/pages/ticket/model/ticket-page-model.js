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

export const MOCK_WORK_RESULT =
    "Проведен профилактический визит, выполнена базовая очистка рабочих поверхностей и узлов. Проверены ключевые элементы, дефектов не выявлено, устройство работает стабильно.";

export const MOCK_TICKET_HISTORY = [
    {
        id: "closed",
        title: "Закрыл тикет",
        date: "14.02.25",
        time: "14:40",
        icon: ticketClosedIcon,
    },
    {
        id: "worksDone",
        title: "Завершил работы",
        date: "14.02.25",
        time: "14:05",
        icon: ticketDoneIcon,
    },
    {
        id: "inWork",
        title: "Начал работы",
        date: "14.02.25",
        time: "12:20",
        icon: ticketInWorkIcon,
    },
    {
        id: "assigned",
        title: "Назначил",
        date: "14.02.25",
        time: "10:30",
        icon: ticketAssignedIcon,
    },
];
