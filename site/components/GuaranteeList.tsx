"use client";

interface GuaranteeListProps {
  items: string[];
}

export default function GuaranteeList({ items }: GuaranteeListProps) {
  return (
    <ol className="space-y-6">
      {items.map((item, i) => (
        <li key={i} className="flex items-start gap-5">
          <div className="w-[3px] bg-[#E0561C] self-stretch rounded-full flex-shrink-0 mt-1" />
          <p className="text-[22px] font-[700] tracking-[-0.02em] text-[#0F1B2D] leading-snug">
            {item}
          </p>
        </li>
      ))}
    </ol>
  );
}
