"use client";

interface CTABandProps {
  headline: string;
  sub?: string;
  cta: string;
  href: string;
  note?: string;
}

export default function CTABand({ headline, sub, cta, href, note }: CTABandProps) {
  return (
    <section className="bg-[#0F1B2D] py-20 px-6">
      <div className="max-w-3xl mx-auto text-center">
        <h2 className="text-[36px] md:text-[48px] font-[800] tracking-[-0.03em] text-white leading-tight mb-4">
          {headline}
        </h2>
        {sub && (
          <p className="text-[18px] text-[#8A98A8] mb-8 max-w-xl mx-auto leading-relaxed">
            {sub}
          </p>
        )}
        <a
          href={href}
          className="inline-block bg-[#E0561C] text-white text-[15px] font-[700] px-8 py-4 rounded-md hover:bg-[#c94d19] transition-colors duration-150"
        >
          {cta}
        </a>
        {note && (
          <p className="text-[13px] text-[#5D6E82] mt-5">{note}</p>
        )}
      </div>
    </section>
  );
}
