"use client";

interface TrackCardProps {
  audience: string;
  headline: string;
  body: string;
  cta: string;
  href: string;
}

export default function TrackCard({ audience, headline, body, cta, href }: TrackCardProps) {
  return (
    <a
      href={href}
      className="group block border border-[#C3CEDA] rounded-lg p-8 bg-white hover:border-[#E0561C] hover:shadow-sm transition-all duration-200"
    >
      <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-3">
        {audience}
      </p>
      <h3 className="text-[20px] font-[700] tracking-[-0.02em] text-[#0F1B2D] leading-snug mb-3">
        {headline}
      </h3>
      <p className="text-[14px] text-[#5D6E82] leading-relaxed mb-6">{body}</p>
      <span className="inline-flex items-center gap-2 text-[13px] font-[700] text-[#E0561C] group-hover:underline">
        {cta} &rarr;
      </span>
    </a>
  );
}
