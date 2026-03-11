import { NavigationCard } from "../../../../shared/ui/navigation-card";

export function TicketNavigationCard({ value, subtitle, onClick, disabled }) {
  return <NavigationCard value={value} subtitle={subtitle} onClick={onClick} disabled={disabled} />;
}
