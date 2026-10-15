import { Construction } from "lucide-react";

// Unavailable is shown for features whose backend phase has not landed. It
// states plainly what is not yet supported rather than faking an interaction.
export function Unavailable({ title, reason }: { title: string; reason: string }) {
  return (
    <div className="page">
      <div className="unavailable">
        <div className="ic"><Construction size={30} strokeWidth={1.5} /></div>
        <h2>{title} is not available yet</h2>
        <p>{reason}</p>
      </div>
    </div>
  );
}
