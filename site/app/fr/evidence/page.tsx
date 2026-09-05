import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import Link from "next/link";

export const metadata = {
  title: "Preuves · Kinara OS",
  description: "Construit. Deploye. En cours de validation pre-pilote. Le premier pilote reel commence en mars 2027.",
};

export default function FrEvidencePage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[108px]">

        {/* ── Hero ─────────────────────────────────────────────── */}
        <section className="bg-white px-8 pt-20 pb-16">
          <div className="max-w-[1440px] mx-auto">
            <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
              Preuves
            </p>
            <h1 className="text-[56px] md:text-[80px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.93] mb-8 max-w-4xl">
              Construit. Deploye. En cours de validation pre-pilote.
            </h1>
            <p className="text-[18px] text-[#5D6E82] leading-relaxed max-w-2xl mb-10">
              Deux locataires deployes sur la meme infrastructure prete pour la production. L'infrastructure est disponible pour demonstration avec des donnees synthetiques.
            </p>

            {/* Mars 2027 callout */}
            <div className="inline-flex items-start gap-5 bg-[#F5F8FB] border-l-4 border-[#E0561C] px-7 py-5 max-w-2xl">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.16em] text-[#E0561C] mb-2">
                  Pilote reel · Mars 2027
                </p>
                <p className="text-[16px] text-[#0F1B2D] leading-relaxed">
                  Kinara OS est actuellement en cours de validation pre-pilote avec des donnees synthetiques dans les environnements Klinova et Village Health Access. Le premier pilote reel est prevu pour mars 2027.
                </p>
              </div>
            </div>
          </div>
        </section>

        {/* ── Village Health Access ─────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-20">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <div className="flex items-start gap-4 mb-8">
                  <div className="w-1 flex-shrink-0 bg-[#10B981] mt-1" style={{ height: "52px" }} />
                  <div>
                    <p className="text-[12px] font-[700] uppercase tracking-[0.14em] text-[#10B981]">
                      Village Health Access &nbsp;&middot;&nbsp; Operateur non lucratif
                    </p>
                    <p className="text-[14px] text-[#8A98A8]">Locataire deploye &middot; Kinara OS</p>
                  </div>
                </div>
                <h2 className="text-[36px] md:text-[48px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  Un operateur deploye en cours de preparation pour la validation sur le terrain.
                </h2>
                <div className="text-[17px] text-[#5D6E82] leading-relaxed space-y-4 mb-10">
                  <p>
                    Donnees synthetiques. Workflows reels. Infrastructure prete pour la production. Gestion des stocks via WhatsApp par des agents de terrain nommes, dossiers structures des la premiere saisie, rapports bailleurs tires comme des requetes sur la meme base de donnees operationnelle.
                  </p>
                  <p>
                    Le premier pilote communautaire de sante au Togo commence en mars 2027.
                  </p>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-6">
                {[
                  { n: "40",  l: "services de sante", sub: "dans le domaine" },
                  { n: "0",   l: "etapes d'export manuel", sub: "par conception" },
                  { n: "72h", l: "d'enregistrements hors ligne", sub: "mis en file d'attente a la peripherie" },
                  { n: "1",   l: "locataire deploye", sub: "meme infrastructure que tous les futurs locataires" },
                ].map((s) => (
                  <div key={s.n} className="bg-white p-8">
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
        <section className="bg-white px-8 py-20">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <div className="flex items-start gap-4 mb-8">
                  <div className="w-1 flex-shrink-0 bg-[#E0561C] mt-1" style={{ height: "52px" }} />
                  <div>
                    <p className="text-[12px] font-[700] uppercase tracking-[0.14em] text-[#E0561C]">
                      Klinova
                    </p>
                    <p className="text-[14px] text-[#8A98A8]">Locataire deploye &middot; Kinara OS</p>
                  </div>
                </div>
                <h2 className="text-[36px] md:text-[48px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  Un operateur deploye en cours de preparation pour la validation sur le terrain.
                </h2>
                <div className="text-[17px] text-[#5D6E82] leading-relaxed space-y-4 mb-10">
                  <p>
                    Klinova est le principal locataire commercial. Il connecte cliniques, hopitaux, assureurs, pharmacies, medecins et reseaux de livraison dans un systeme de sante coordonne. Triage patient, ordonnances numeriques, orientation des referencements et paiements des partenaires fonctionnent tous sur la meme infrastructure Kinara OS.
                  </p>
                  <p>
                    Chaque produit que Klinova deploie utilise le meme modele de gouvernance, le meme journal d'audit et les memes controles d'acces mesures que tout futur locataire herite des le premier jour.
                  </p>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-6">
                {[
                  { n: "4",   l: "types de partenaires", sub: "cliniques, pharmacies, medecins, livraison" },
                  { n: "48h", l: "cycle de paiement", sub: "facture et regle par consultation" },
                  { n: "0",   l: "frais d'installation", sub: "pour les partenaires pilotes" },
                  { n: "1",   l: "locataire deploye", sub: "meme infrastructure que tous les futurs locataires" },
                ].map((s) => (
                  <div key={s.l} className="bg-[#F5F8FB] p-8">
                    <p className="text-[48px] font-[800] tracking-[-0.05em] leading-none mb-1" style={{ color: "#E0561C" }}>{s.n}</p>
                    <p className="text-[15px] font-[700] text-[#0F1B2D]">{s.l}</p>
                    <p className="text-[12px] text-[#8A98A8] mt-0.5">{s.sub}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ── Verifiable ───────────────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-20">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Ce qui est verifiable</p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-6">
                  Chaque chiffre trace jusqu'a sa source.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">
                  Un bailleur ou un auditeur peut recalculer tout chiffre rapporte en reexecutant la meme requete sur la meme base de donnees operationnelle. Rien n'est pre-agrege dans une couche de reporting distincte.
                </p>
              </div>
              <ul className="space-y-0">
                {[
                  "Chaque dossier de visite attribue a un agent de sante nomme dans un etablissement identifie",
                  "Chaque entree de stock horodatee au moment de la saisie, pas du telechargement",
                  "Chaque requete interdomaine ayant produit un resultat logistique figure dans le journal d'audit",
                  "Chaque chiffre d'un rapport peut etre recalcule en reexecutant la meme requete",
                ].map((item, i) => (
                  <li key={i} className="flex items-start gap-5 px-6 py-5 bg-white mb-2 last:mb-0">
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
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Demander une demonstration</p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-6">
                  Nous vous montrerons le locataire, les enregistrements et le journal d'audit.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">
                  L'infrastructure est deployee et disponible pour demonstration avec des donnees synthetiques. Un entretien dure 45 minutes.
                </p>
              </div>
              <div className="flex flex-col gap-4 items-start">
                <Link href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[15px] font-[700] px-9 py-4 hover:bg-[#c94d19] transition-colors">
                  Demander un entretien
                </Link>
                <p className="text-[13px] text-[#3A4F68] mt-2">Kinara OS est concu et detenu par Klinova &nbsp;&middot;&nbsp; kinaraos.com</p>
              </div>
            </div>
          </div>
        </section>

      </main>
      <Footer lang="fr" />
    </>
  );
}
