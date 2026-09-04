"use client";

interface DomainCardProps {
  name: string;
  services: number;
  steward: string;
  description: string;
  href: string;
  icon: React.ReactNode;
}

export default function DomainCard({ name, services, steward, description, href, icon }: DomainCardProps) {
  return (
    <a
      href={href}
      className="group block border border-[#C3CEDA] rounded-lg p-6 bg-white hover:border-[#E0561C] hover:shadow-sm transition-all duration-200"
    >
      <div className="mb-4 text-[#0F1B2D]">{icon}</div>
      <h3 className="text-[20px] font-[800] tracking-[-0.02em] text-[#0F1B2D] mb-1">
        {name}
      </h3>
      <p className="text-[12px] font-[600] uppercase tracking-[0.1em] text-[#8A98A8] mb-3">
        {services} services · {steward}
      </p>
      <p className="text-[14px] text-[#5D6E82] leading-relaxed mb-4">{description}</p>
      <span className="text-[13px] font-[600] text-[#E0561C] group-hover:underline">
        See the domain &rarr;
      </span>
    </a>
  );
}
