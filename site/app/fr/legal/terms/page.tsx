import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export const metadata = { title: "Conditions d'utilisation · Kinara OS" };

export default function FrTermsPage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[100px]">
        <section className="max-w-[1440px] mx-auto px-8 pt-16 pb-20">
          <div className="max-w-2xl">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">Mentions légales</p>
            <h1 className="text-[40px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-tight mb-8">Conditions d&apos;utilisation</h1>
            <div className="text-[#5D6E82] leading-relaxed space-y-6 text-[16px]">
              <p className="text-[14px] text-[#8A98A8]">Dernière mise à jour : 1er septembre 2026</p>
              <p>Ces conditions régissent l&apos;utilisation du site kinaraos.com, exploité par Klinova Health LLC. En accédant à ce site, vous acceptez ces conditions.</p>
              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Utilisation du site</h2>
              <p>Ce site fournit des informations sur la plateforme Kinara OS. Vous pouvez lire et partager le contenu à des fins d&apos;information. Vous ne pouvez pas reproduire, distribuer ou créer des œuvres dérivées du contenu sans l&apos;autorisation écrite préalable de Klinova.</p>
              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Absence de garantie</h2>
              <p>Le contenu de ce site est fourni à titre informatif uniquement. Klinova ne fait aucune représentation ni garantie, expresse ou implicite, concernant l&apos;exactitude ou l&apos;exhaustivité de tout contenu.</p>
              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Droit applicable</h2>
              <p>Ces conditions sont régies par les lois du Commonwealth de Virginie, États-Unis d&apos;Amérique.</p>
            </div>
          </div>
        </section>
      </main>
      <Footer lang="fr" />
    </>
  );
}
