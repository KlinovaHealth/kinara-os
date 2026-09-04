import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export const metadata = { title: "Politique de confidentialité · Kinara OS" };

export default function FrPrivacyPage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[100px]">
        <section className="max-w-[1440px] mx-auto px-8 pt-16 pb-20">
          <div className="max-w-2xl">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">Mentions légales</p>
            <h1 className="text-[40px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-tight mb-8">Politique de confidentialité</h1>
            <div className="text-[#5D6E82] leading-relaxed space-y-6 text-[16px]">
              <p className="text-[14px] text-[#8A98A8]">Dernière mise à jour : 1er septembre 2026</p>
              <p>Cette politique décrit la manière dont Klinova Health LLC («&nbsp;Klinova&nbsp;», «&nbsp;nous&nbsp;») traite les données personnelles collectées via le site kinaraos.com.</p>
              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Ce que nous collectons</h2>
              <p>Lorsque vous soumettez le formulaire de contact, nous collectons votre nom, votre organisation, votre adresse électronique et les informations que vous fournissez sur votre problème de coordination. Nous utilisons ces informations uniquement pour répondre à votre demande d&apos;entretien.</p>
              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Comment nous l&apos;utilisons</h2>
              <p>Les soumissions du formulaire de contact servent à répondre aux demandes d&apos;entretien. Nous ne vendons, ne partageons ni ne transférons ces informations à des tiers, sauf si nécessaire pour délivrer notre réponse.</p>
              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Données des locataires</h2>
              <p>Cette politique ne couvre que le site web marketing kinaraos.com. Si vous êtes un locataire ou opérateur Kinara OS, les conditions de gouvernance des données de votre contrat de location régissent le traitement des données opérationnelles.</p>
              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Contact</h2>
              <p>Les questions relatives à cette politique peuvent être adressées à Klinova Health LLC via le formulaire de contact de ce site.</p>
            </div>
          </div>
        </section>
      </main>
      <Footer lang="fr" />
    </>
  );
}
