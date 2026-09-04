import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import Link from "next/link";

export const metadata = { title: "Pour qui · Kinara OS" };

const OPERATORS = [
  { type: "Cliniques et réseaux hospitaliers", color: "#10B981", desc: "Partagez des signaux de référencement, le routage des patients et les niveaux de stock entre établissements sans centraliser les dossiers. Chaque clinique reste propriétaire de ses données." },
  { type: "Assureurs", color: "#E0561C", desc: "Recevez des signaux de vérification de sinistres sur des dossiers de santé que vous ne détenez pas. La vérification franchit la frontière; le dossier reste dans la clinique." },
  { type: "Opérateurs logistiques et transport", color: "#3B82F6", desc: "Recevez des signaux de demande des systèmes de santé et d'agriculture. Acheminez les véhicules sur des besoins confirmés, pas des estimations. Chaque expédition est consignée." },
  { type: "Agribusiness et coopératives", color: "#F59E0B", desc: "Connectez les données de récolte confirmées au routage vers les marchés et à l'approvisionnement en intrants. Le dossier reste chez l'agriculteur; le signal atteint l'acheteur." },
  { type: "Autorités portuaires et opérateurs maritimes", color: "#0EA5E9", desc: "Coordonnez la mainlevée des marchandises avec la logistique intérieure en temps réel. Deux opérateurs, un signal, aucun dossier partagé, piste d'audit complète." },
  { type: "Organismes publics et régulateurs", color: "#8B5CF6", desc: "Coordonnez entre agences sans créer de référentiel central. Chaque organisme conserve la garde. Chaque requête transfrontalière est attribuée et consignée." },
];

const HOW_IT_WORKS = [
  { heading: "Chaque opérateur conserve ses propres systèmes.", body: "Les dossiers de santé restent dans les systèmes de santé. Les données logistiques restent dans les systèmes logistiques. Kinara OS ne les déplace ni ne les copie." },
  { heading: "Les requêtes franchissent les frontières sous politique.", body: "Lorsqu'une question intersystème est autorisée, Kinara Core vérifie la politique, exécute la jointure avec un minimum de divulgation et consigne l'entrée d'audit." },
  { heading: "Chaque action est attribuée.", body: "Qui a demandé, ce qu'il a demandé, ce qui a été retourné. Tout figure dans le journal d'audit. Aucune des deux parties ne peut le modifier." },
  { heading: "Fonctionne depuis la périphérie.", body: "Des agents de terrain sur des téléphones basiques, hors ligne pendant 72 heures. Les enregistrements se mettent en file d'attente et se synchronisent au retour de la connectivité." },
];

export default function FrGovernmentsPage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[108px]">

        {/* ── Hero ─────────────────────────────────────────────── */}
        <section className="bg-white px-8 pt-20 pb-16 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Pour qui</p>
            <h1 className="text-[56px] md:text-[80px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.93] mb-8 max-w-4xl">
              Conçu pour tout opérateur dont les données doivent franchir une frontière.
            </h1>
            <p className="text-[18px] text-[#5D6E82] leading-relaxed max-w-2xl mb-10">
              Kinara OS connecte cliniques, assureurs, transporteurs, agribusiness, opérateurs portuaires et organismes publics dans un système coordonné. Aucun opérateur ne cède les données qu'il possède. Aucun référentiel central n'est créé.
            </p>
            <div className="flex flex-wrap gap-4">
              <Link href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-7 py-3.5 hover:bg-[#c94d19] transition-colors">
                Demander un entretien
              </Link>
              <Link href="/fr/platform" className="inline-block border border-[#C3CEDA] text-[#0F1B2D] text-[14px] font-[600] px-7 py-3.5 hover:border-[#0F1B2D] transition-colors">
                Voir l'architecture
              </Link>
            </div>
          </div>
        </section>

        {/* ── Le problème ──────────────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-20 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Le problème</p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  Les données existent. La connexion n'existe pas.
                </h2>
                <div className="text-[17px] text-[#5D6E82] leading-relaxed space-y-4">
                  <p>
                    Une rupture de stock en santé est aussi un échec logistique. La mainlevée d'un conteneur au port est aussi un signal d'acheminement pour le transport intérieur. Une récolte confirmée est aussi une prévision de demande pour les fournisseurs d'intrants agricoles.
                  </p>
                  <p>
                    Chacun de ces problèmes est soluble avec des données qui existent déjà dans des organisations séparées. Le problème est de franchir la frontière légalement, rapidement et avec un enregistrement de ce qui s'est passé.
                  </p>
                  <p>
                    Cela s'applique également à un réseau de cliniques privées, un assureur, un transporteur et une autorité portuaire. L'échec de coordination n'est pas un problème de gouvernement. C'est un problème d'infrastructure.
                  </p>
                </div>
              </div>
              <div className="relative p-10" style={{ background: "#080F1E" }}>
                <div className="absolute inset-0 dot-grid pointer-events-none opacity-50" />
                <div className="relative">
                  <div className="w-8 h-px bg-[#E0561C] mb-8" />
                  <p className="text-[28px] font-[800] text-white leading-snug tracking-[-0.03em] mb-8">
                    Les grilles africaines ne sont pas déconnectées parce que les données n'existent pas. Elles le sont parce qu'il n'existe pas de façon gouvernée de les faire franchir une frontière organisationnelle.
                  </p>
                  <p className="text-[14px] font-[600] text-[#E0561C] uppercase tracking-[0.12em]">Kinara OS est ce passage.</p>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* ── Types d'opérateurs ────────────────────────────────── */}
        <section className="bg-white px-8 py-20 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="mb-12">
              <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-4">Types d'opérateurs</p>
              <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] max-w-2xl">
                Privé, public, ou les deux. L'infrastructure est la même.
              </h2>
            </div>
            <div className="grid md:grid-cols-3 gap-px bg-[#C3CEDA]">
              {OPERATORS.map((op) => (
                <div key={op.type} className="bg-white p-8 group card-lift cursor-default">
                  <div className="w-8 h-1 mb-5 rounded-full" style={{ background: op.color }} />
                  <h3 className="text-[18px] font-[800] text-[#0F1B2D] tracking-[-0.02em] mb-3">{op.type}</h3>
                  <p className="text-[14px] text-[#5D6E82] leading-relaxed">{op.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ── Comment ça marche ─────────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-20 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Comment ça marche</p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-6">
                  Une couche de jointure gouvernée, pas un référentiel central.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-8">
                  Aucun opérateur ne cède la garde de ce qu'il possède. Ce qui change, c'est qu'une question vérifiée peut désormais franchir une frontière, sous politique, avec un enregistrement immuable de ce qui s'est passé.
                </p>
                <Link href="/fr/governance" className="text-[14px] font-[700] text-[#E0561C] hover:text-[#c94d19] transition-colors">
                  Découvrir le modèle de gouvernance &rarr;
                </Link>
              </div>
              <div className="space-y-0 border border-[#C3CEDA]">
                {HOW_IT_WORKS.map((item, i) => (
                  <div key={i} className="bg-white px-7 py-6 border-b border-[#C3CEDA] last:border-0">
                    <p className="text-[16px] font-[700] text-[#0F1B2D] mb-1.5">{item.heading}</p>
                    <p className="text-[14px] text-[#5D6E82] leading-relaxed">{item.body}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ── CTA ──────────────────────────────────────────────── */}
        <section className="relative px-8 py-24 overflow-hidden" style={{ background: "radial-gradient(ellipse 60% 80% at 30% 50%, rgba(224,86,28,0.15) 0%, transparent 60%), #080F1E" }}>
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
          <div className="relative max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-center">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Commencer une conversation</p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-6">
                  Apportez-nous le problème qui franchit deux organisations.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">Un entretien dure 45 minutes. Nous vous montrerons le système en fonctionnement, pas des diapositives. Public ou privé, la conversation est la même.</p>
              </div>
              <div className="flex flex-col gap-4 items-start">
                <Link href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[15px] font-[700] px-9 py-4 hover:bg-[#c94d19] transition-colors">
                  Demander un entretien
                </Link>
                <Link href="/fr/evidence" className="inline-block border border-[#3A4F68] text-[#8A98A8] text-[15px] font-[600] px-9 py-4 hover:border-[#8A98A8] hover:text-white transition-colors">
                  Voir qui l'utilise déjà
                </Link>
              </div>
            </div>
          </div>
        </section>

      </main>
      <Footer lang="fr" />
    </>
  );
}
