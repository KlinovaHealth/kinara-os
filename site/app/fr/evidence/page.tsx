import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import Link from "next/link";

export const metadata = { title: "Preuves · Kinara OS" };

export default function FrEvidencePage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[108px]">

        {/* ── Hero ─────────────────────────────────────────────── */}
        <section className="bg-white px-8 pt-20 pb-16 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
              Preuves
            </p>
            <h1 className="text-[56px] md:text-[80px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.93] mb-8 max-w-4xl">
              Pas un document de présentation. Deux locataires actifs.
            </h1>
            <p className="text-[18px] text-[#5D6E82] leading-relaxed max-w-2xl">
              Deux opérateurs, deux locations réelles, la même infrastructure gouvernée que tout futur locataire utilisera. Rien de mis en scène, rien en bac à sable.
            </p>
          </div>
        </section>

        {/* ── Village Health Access ─────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-20 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <div className="flex items-start gap-4 mb-8">
                  <div className="w-1 flex-shrink-0 bg-[#10B981] mt-1" style={{ height: "52px" }} />
                  <div>
                    <p className="text-[12px] font-[700] uppercase tracking-[0.14em] text-[#10B981]">
                      Village Health Access &nbsp;&middot;&nbsp; Opérateur non lucratif
                    </p>
                    <p className="text-[14px] text-[#8A98A8]">Location active &middot; Kinara OS</p>
                  </div>
                </div>
                <h2 className="text-[36px] md:text-[48px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  Un opérateur actif, sur une location active.
                </h2>
                <div className="text-[17px] text-[#5D6E82] leading-relaxed space-y-4 mb-10">
                  <p>
                    Un réseau de cliniques communautaires en activité. De vrais patients. De vrais dossiers. Stock saisi via WhatsApp par des agents de terrain identifiés. Rapports tirés directement de la base de données opérationnelle. Aucun export manuel, aucun tableur.
                  </p>
                  <p>
                    Le rapport du bailleur est une requête sur les mêmes données que celles utilisées par la clinique pour gérer sa journée.
                  </p>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-px bg-[#C3CEDA] border border-[#C3CEDA]">
                {[
                  { n: "40",  l: "services de santé", sub: "dans leur domaine" },
                  { n: "0",   l: "étapes d'export manuel", sub: "par rapport de bailleur" },
                  { n: "72h", l: "d'enregistrements hors ligne", sub: "mis en file d'attente à la périphérie" },
                  { n: "1",   l: "location active", sub: "même infrastructure que tous les futurs locataires" },
                ].map((s) => (
                  <div key={s.n} className="bg-[#F5F8FB] p-8">
                    <p className="text-[48px] font-[800] tracking-[-0.05em] leading-none mb-1" style={{ color: "#10B981" }}>{s.n}</p>
                    <p className="text-[15px] font-[700] text-[#0F1B2D]">{s.l}</p>
                    <p className="text-[12px] text-[#8A98A8] mt-0.5">{s.sub}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ── Klinova ──────────────────────────────────────────── */}
        <section className="bg-white px-8 py-20 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <div className="flex items-start gap-4 mb-8">
                  <div className="w-1 flex-shrink-0 bg-[#E0561C] mt-1" style={{ height: "52px" }} />
                  <div>
                    <p className="text-[12px] font-[700] uppercase tracking-[0.14em] text-[#E0561C]">
                      Klinova &nbsp;&middot;&nbsp; Opérateur commercial
                    </p>
                    <p className="text-[14px] text-[#8A98A8]">Location active &middot; Kinara OS</p>
                  </div>
                </div>
                <h2 className="text-[36px] md:text-[48px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  Un opérateur actif, sur une location active.
                </h2>
                <div className="text-[17px] text-[#5D6E82] leading-relaxed space-y-4 mb-10">
                  <p>
                    Klinova est le principal locataire commercial. Il connecte cliniques, hôpitaux, assureurs, pharmacies, médecins et réseaux de livraison dans un système de santé coordonné. Triage patient, ordonnances numériques, orientation des référencements et paiements des partenaires fonctionnent tous sur la même infrastructure Kinara OS.
                  </p>
                  <p>
                    Chaque produit que Klinova déploie utilise le même modèle de gouvernance, le même journal d'audit et les mêmes contrôles d'accès mesurés que tout futur locataire hérite dès le premier jour.
                  </p>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-px bg-[#C3CEDA] border border-[#C3CEDA]">
                {[
                  { n: "4",   l: "types de partenaires", sub: "cliniques, pharmacies, médecins, livraison" },
                  { n: "48h", l: "cycle de paiement", sub: "facturé et réglé par consultation" },
                  { n: "0",   l: "frais d'installation", sub: "pour les partenaires pilotes" },
                  { n: "1",   l: "location active", sub: "même infrastructure que tous les futurs locataires" },
                ].map((s) => (
                  <div key={s.l} className="bg-white p-8">
                    <p className="text-[48px] font-[800] tracking-[-0.05em] leading-none mb-1" style={{ color: "#E0561C" }}>{s.n}</p>
                    <p className="text-[15px] font-[700] text-[#0F1B2D]">{s.l}</p>
                    <p className="text-[12px] text-[#8A98A8] mt-0.5">{s.sub}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ── Vérifiable ───────────────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-20 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Ce qui est vérifiable</p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-6">
                  Chaque chiffre tracé jusqu'à sa source.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">
                  Un bailleur ou un auditeur peut recalculer tout chiffre rapporté en réexécutant la même requête sur la même base de données opérationnelle. Rien n'est pré-agrégé dans une couche de reporting distincte.
                </p>
              </div>
              <ul className="space-y-0 border border-[#C3CEDA]">
                {[
                  "Chaque dossier de visite attribué à un agent de santé nommé dans un établissement identifié",
                  "Chaque entrée de stock horodatée au moment de la saisie, pas du téléchargement",
                  "Chaque requête interdomaine ayant produit un résultat logistique figure dans le journal d'audit",
                  "Chaque chiffre d'un rapport peut être recalculé en réexécutant la même requête",
                ].map((item, i) => (
                  <li key={i} className="flex items-start gap-5 px-6 py-5 border-b border-[#C3CEDA] last:border-0 bg-white">
                    <span className="flex-shrink-0 w-5 h-5 mt-0.5 rounded-full bg-[#E0561C] flex items-center justify-center">
                      <svg width="10" height="8" viewBox="0 0 10 8" fill="none"><path d="M1 4l2.5 2.5L9 1" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/></svg>
                    </span>
                    <p className="text-[16px] text-[#0F1B2D] leading-snug">{item}</p>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </section>

        {/* ── CTA ──────────────────────────────────────────────── */}
        <section className="relative px-8 py-24 overflow-hidden" style={{ background: "radial-gradient(ellipse 60% 80% at 30% 50%, rgba(224,86,28,0.15) 0%, transparent 60%), #080F1E" }}>
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
          <div className="relative max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-center">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Demander une démonstration</p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-6">
                  Nous vous montrerons le locataire, les enregistrements et le journal d'audit.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">Tout est en direct, rien n'est mis en scène. Un entretien dure 45 minutes.</p>
              </div>
              <div className="flex flex-col gap-4 items-start">
                <Link href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[15px] font-[700] px-9 py-4 hover:bg-[#c94d19] transition-colors">
                  Demander un entretien
                </Link>
                <p className="text-[13px] text-[#3A4F68] mt-2">Kinara OS est conçu et détenu par Klinova &nbsp;&middot;&nbsp; kinaraos.com</p>
              </div>
            </div>
          </div>
        </section>

      </main>
      <Footer lang="fr" />
    </>
  );
}
