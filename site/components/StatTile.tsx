"use client";

interface StatTileProps {
  figure: string;
  label: string;
  source?: string;
}

export default function StatTile({ figure, label, source }: StatTileProps) {
  return (
    <div className="border border-[#C3CEDA] rounded-lg px-6 py-5 bg-white">
      <p className="text-[42px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-none mb-1">
        {figure}
      </p>
      <p className="text-[14px] font-[500] text-[#5D6E82]">{label}</p>
      {source && (
        <p className="text-[12px] text-[#8A98A8] mt-2">{source}</p>
      )}
    </div>
  );
}
