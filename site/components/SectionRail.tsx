"use client";

interface SectionRailProps {
  label: string;
  description: string;
}

export default function SectionRail({ label, description }: SectionRailProps) {
  return (
    <div className="flex items-start gap-4 mb-12">
      <div className="w-[3px] bg-[#E0561C] self-stretch rounded-full flex-shrink-0" />
      <div>
        <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-1">
          {label}
        </p>
        <p className="text-[14px] text-[#5D6E82] italic">{description}</p>
      </div>
    </div>
  );
}
