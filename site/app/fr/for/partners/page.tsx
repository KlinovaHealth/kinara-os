import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import Link from "next/link";

export const metadata = { title: "Pour les partenaires de santé · Klinova" };

const PARTNER_TYPES = [
  { type: "Cliniques et hôpitaux", tag: "Référencements + tableau de bord", color: "#10B981", desc: "Recevez des référencements patients pré-triés prêts pour une prise en charge en présentiel. Les dossiers numériques des patients arrivent avant qu'ils franchissent la porte." },
  { type: "Pharmacies", tag: "Ordonnance numérique + livraison", color: "#E0561C", desc: "Recevez des ordonnances numériques directement des médecins Klinova. Zéro papier, notification instantanée, dispensation plus rapide." },
  { type: "Médecins et infirmiers", tag: "Revenus de téléconsultation", color: "#3B82F6", desc: "Consultez des patients par chat, voix ou vidéo selon votre propre planning. Klinova gère le triage, les dossiers et le paiement." },
  { type: "Livraison et transport", tag: "Livraison du dernier kilomètre", color: "#F59E0B", desc: "Livrez des médicaments des pharmacies partenaires aux patients. Rejoignez le réseau et recevez des demandes de livraison régulières et vérifiées." },
  { type: "Compagnies d'assurance", tag: "Vérification des sinistres", color: "#8B5CF6", desc: "Recevez des signaux de vérification de sinistres sur des dossiers de santé que vous ne détenez pas. La vérification franchit la frontière; le dossier reste dans la clinique." },
  { type: "Gouvernements et ONG", tag: "Santé des populations", color: "#0EA5E9", desc: "Coordonnez les programmes de santé publique entre réseaux de cliniques, chaînes d'approvisionnement et agents de santé communautaires. Signaux agrégés, pas de dossiers bruts." },
];

const DASHBOARD_FEATURES = [
  { title: "File d'attente de référencements patients", desc: "Consultez en temps réel les référencements Klinova entrants. Acceptez ou reportez en un clic. Dossier patient joint." },
  { title: "Gestion des rendez-vous", desc: "Calendrier de réservation complet lié à votre capacité réelle. Les patients voient votre disponibilité en temps réel." },
  { title: "Dossiers de santé numériques", desc: "Accédez à l'historique du patient, aux consultations antérieures et aux ordonnances. Aucun formulaire papier." },
  { title: "Module pharmacie et ordonnances", desc: "Émettez des ordonnances numériques qui s'acheminent automatiquement vers la pharmacie partenaire la plus proche." },
  { title: "Analyses et rapports", desc: "Suivez les consultations, référencements, revenus et résultats patients par jour, semaine ou mois." },
  { title: "Paiements et facturation", desc: "Klinova gère la facturation des patients et vous verse votre part dans les 48 heures suivant chaque consultation." },
];

export default function FrPartnersPage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[108px]">

        {/* ── Hero ─────────────────────────────────────────────── */}
        <section className="bg-white px-8 pt-20 pb-0">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-end pb-16 border-b border-[#C3CEDA]">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Pour les partenaires de santé</p>
                <h1 className="text-[56px] md:text-[72px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.93] mb-8">
                  Rejoignez la grille de santé africaine.
                </h1>
                <p className="text-[18px] text-[#5D6E82] leading-relaxed mb-10 max-w-lg">
                  Les cliniques, pharmacies, médecins, employeurs et prestataires de livraison rejoignent le réseau Klinova et reçoivent des référencements patients vérifiés.
                </p>
                <div className="flex flex-wrap gap-4">
                  <Link href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-7 py-3.5 hover:bg-[#c94d19] transition-colors">
                    Devenir partenaire
                  </Link>
                  <Link href="/fr/contact" className="inline-block border border-[#C3CEDA] text-[#0F1B2D] text-[14px] font-[600] px-7 py-3.5 hover:border-[#0F1B2D] transition-colors">
                    Parler à notre équipe
                  </Link>
                </div>
              </div>
              <div className="space-y-0 border border-[#C3CEDA]">
                {[
                  { label: "Aucun frais d'installation", sub: "pour les partenaires pilotes" },
                  { label: "Référencements patients instantanés", sub: "dès le premier jour" },
                  { label: "Modèle de partage des revenus", sub: "payé dans les 48 heures" },
                ].map((b, i) => (
                  <div key={i} className="flex items-center gap-5 px-6 py-5 border-b border-[#C3CEDA] last:border-0">
                    <div className="flex-shrink-0 w-2 h-2 rounded-full" style={{ background: "#E0561C" }} />
                    <div>
                      <p className="text-[16px] font-[700] text-[#0F1B2D]">{b.label}</p>
                      <p className="text-[13px] text-[#8A98A8]">{b.sub}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ── Types de partenaires ──────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-20 border-y border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="mb-12">
              <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-4">Types de partenaires</p>
              <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95]">
                Que vous ayez une clinique, une pharmacie, un laboratoire ou une moto, il y a une place pour vous.
              </h2>
            </div>
            <div className="grid md:grid-cols-3 gap-px bg-[#C3CEDA]">
              {PARTNER_TYPES.map((p) => (
                <div key={p.type} className="bg-white p-8 group card-lift cursor-default">
                  <div className="flex items-start gap-4 mb-5">
                    <div className="w-1 flex-shrink-0 rounded-full" style={{ background: p.color, minHeight: "48px" }} />
                    <div className="flex-1">
                      <h3 className="text-[20px] font-[800] text-[#0F1B2D] tracking-[-0.02em] mb-1">{p.type}</h3>
                      <span className="inline-block text-[11px] font-[700] uppercase tracking-[0.12em] px-2.5 py-1 mb-4" style={{ background: p.color + "18", color: p.color }}>{p.tag}</span>
                      <p className="text-[15px] text-[#5D6E82] leading-relaxed">{p.desc}</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ── Tableau de bord ───────────────────────────────────── */}
        <section className="bg-white px-8 py-20">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div className="md:sticky md:top-32">
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Tableau de bord partenaire clinique</p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-6">
                  Tout ce dont votre clinique a besoin, en un seul endroit.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-8">
                  Le tableau de bord partenaire de Klinova est conçu autour du fonctionnement réel des cliniques: rendez-vous, référencements, ordonnances et performances, tout en un seul écran.
                </p>
                <Link href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-7 py-3.5 hover:bg-[#c94d19] transition-colors">
                  Devenir partenaire
                </Link>
              </div>
              <div className="space-y-0 border border-[#C3CEDA]">
                {DASHBOARD_FEATURES.map((f, i) => (
                  <div key={i} className="px-7 py-6 border-b border-[#C3CEDA] last:border-0 hover:bg-[#F5F8FB] transition-colors">
                    <div className="flex items-start gap-4">
                      <span className="text-[12px] font-[800] text-[#E0561C] flex-shrink-0 mt-0.5 w-5 text-right">{String(i + 1).padStart(2, "0")}</span>
                      <div>
                        <p className="text-[16px] font-[700] text-[#0F1B2D] mb-1">{f.title}</p>
                        <p className="text-[14px] text-[#5D6E82] leading-relaxed">{f.desc}</p>
                      </div>
                    </div>
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
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Prêt à rejoindre</p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-6">
                  Candidatez comme partenaire pilote. Aucun frais d'installation.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">Les partenaires pilotes ne paient aucun frais d'installation. Vous recevez des référencements dès le premier jour et percevez une part de chaque consultation facturée via le réseau.</p>
              </div>
              <div className="flex flex-col gap-4 items-start">
                <Link href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[15px] font-[700] px-9 py-4 hover:bg-[#c94d19] transition-colors">
                  Devenir partenaire
                </Link>
                <Link href="/fr/contact" className="inline-block border border-[#3A4F68] text-[#8A98A8] text-[15px] font-[600] px-9 py-4 hover:border-[#8A98A8] hover:text-white transition-colors">
                  Parler à notre équipe
                </Link>
                <p className="text-[13px] text-[#3A4F68] mt-2">Klinova fonctionne sur Kinara OS &nbsp;&middot;&nbsp; kinaraos.com</p>
              </div>
            </div>
          </div>
        </section>

      </main>
      <Footer lang="fr" />
    </>
  );
}
