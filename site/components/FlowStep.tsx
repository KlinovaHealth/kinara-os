"use client";

interface FlowStepProps {
  number: number;
  heading: string;
  body: string;
  active?: boolean;
}

export default function FlowStep({ number, heading, body, active = false }: FlowStepProps) {
  return (
    <div
      className={`relative flex items-start gap-5 p-6 rounded-lg border transition-all duration-300 ${
        active
          ? "border-[#E0561C] bg-white shadow-sm"
          : "border-[#C3CEDA] bg-[#F5F8FB]"
      }`}
    >
      <div
        className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center text-[13px] font-[800] transition-colors duration-300 ${
          active ? "bg-[#E0561C] text-white" : "bg-[#C3CEDA] text-[#5D6E82]"
        }`}
      >
        {number}
      </div>
      <div>
        <p className="text-[16px] font-[700] text-[#0F1B2D] mb-1">{heading}</p>
        <p className="text-[14px] text-[#5D6E82] leading-relaxed">{body}</p>
      </div>
    </div>
  );
}
