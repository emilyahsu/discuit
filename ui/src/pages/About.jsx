import React, { useState } from 'react';
import Link from '../components/Link';
import StaticPage from '../components/StaticPage';

const About = () => {
  const faqItems = [
    {
      question: 'How does Seddit make money?',
      answer: (
        <>
          {'We don't. Seddit is funded entirely through donations from our users via our '}
          <a href="https://www.patreon.com/seddit" target="_blank" rel="noreferrer">
            Patreon page
          </a>
          .
        </>
      ),
    },
    {
      question: 'Will Seddit have video uploads?',
      answer: (
        <>
          {'No. While we do plan to support video embeds from platforms like YouTube, '}
          hosting video is highly expensive—but we're quite confident that Seddit can
          handle video embeds from other platforms. Video hosting, on the other hand, is
          expensive, and Seddit does not, nor do we plan to, have native video uploads.
        </>
      ),
    },
    {
      question: 'Is Seddit a federated platform?',
      answer: (
        <>
          {`Seddit is not a federated social platform, and we do not plan to support
          federation in the future. We believe that federation, while a noble goal,
          often leads to a worse user experience and more complex moderation challenges.`}
        </>
      ),
    },
    {
      question: 'Does Seddit have a mobile app?',
      answer: (
        <>
          {`At the moment, Seddit does not have an official iOS or Android app. But we do
          have a Progressive Web App (PWA) that you can install on your phone. To install
          it, visit Seddit on your phone's browser and you should see a prompt to install
          the app. If you don't see the prompt, you can also add it to your home screen
          manually.`}
        </>
      ),
    },
    {
      question: 'How can I contact someone at Seddit?',
      answer: (
        <>
          {'You can reach us at '}
          <a href="mailto:contact@seddit.org">contact@seddit.org</a>
          {'. We try to respond to all emails within 24 hours. If you're reporting a bug,
          please include as much detail as possible, including your browser and operating
          system.'}
        </>
      ),
    },
    {
      question: 'Where can I ask questions about Seddit?',
      answer: (
        <>
          {`If you have a question about Seddit that's not addressed here, feel free to
          ask in our `}
          <Link to="/SedditMeta">SedditMeta</Link>
          {' or '}
          <Link to="/SedditSuggestions">SedditSuggestions</Link>
          {' communities.'}
        </>
      ),
    },
  ];
  const [faqItemOpenedIndex, _setFaqItemOpenedIndex] = useState(null);
  const setFaqItemOpenedIndex = (index) => {
    _setFaqItemOpenedIndex((value) => {
      if (value === index) return null;
      return index;
    });
  };

  const renderFaqItems = () => {
    const elems = faqItems.map((item, index) => {
      const { question, answer } = item;
      const isOpen = faqItemOpenedIndex === index;
      return (
        <div className={'about-faq-item' + (isOpen ? ' is-open' : '')} key={index}>
          <div className="about-faq-question" onClick={() => setFaqItemOpenedIndex(index)}>
            <span>{question}</span>
            <svg
              width="19"
              height="10"
              viewBox="0 0 19 10"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path d="M1 1L9.5 8L17.5 1" stroke="currentColor" strokeWidth="2" />
            </svg>
          </div>
          <div className="about-faq-answer">{answer}</div>
        </div>
      );
    });
    return <>{elems}</>;
  };

  return (
    <StaticPage className="page-about" title="About" noWrap>
      <div className="about-landing">
        <div className="wrap">
          <h1 className="about-heading heading-highlight">
            A social platform by the users, for the users.
          </h1>
          <h2 className="about-subheading">
            Seddit is a non-profit, open-source community discussion platform. It's an alternative
            to traditional social media platforms, focusing on community-driven discussions
            and content moderation.
          </h2>
        </div>
        <div className="squiggly-line"></div>
      </div>
      <div className="about-rest">
        <div className="wrap">
          <div className="about-section about-mission">
            <p>
              Our mission is to build the first large-scale social media platform where the
              interests of the platform are aligned with the interests of the user—a platform, in
              other words, that's built on principles of ethical design. At the heart of these
              principles is the idea of giving users the freedom to choose their online social
              experience as they would prefer.
            </p>
            <p>
              Social media platforms have hitherto done the opposite and taken away what little
              control the users had, as it served those companies' own self-interest, which was and
              still remains, to make as much money as possible, without any regard for, indeed to
              the utter detriment of, the well-being of the user.
            </p>
            {/*<p>
              {`For more information, see the article: `}
              <a href="https://discuit.substack.com" target="_blank" rel="noreferrer">
                {`Why we're building an alternative to Reddit.`}
              </a>
            </p>*/}
          </div>
          <div className="about-section about-highlights">
            <div className="about-highlight">
              <span className="is-bold">No ads. No tracking.</span>
              There are no ads, no forms of affiliate marketing, and no tracking anywhere on
              Seddit. And neither your attention, nor your data, is monetized in any way, shape or
              form.
            </div>
            <div className="about-highlight">
              <span className="is-bold">Enshitification-proof.</span>
              {`Seddit is a non-profit that's funded entirely by its users through donations. The
              lack of a profit motive—and the lack of any shareholders or investors to answer to—is
              essential in keeping this platform completely aligned with the interests and the
              well-being of its users.`}
            </div>
            <div className="about-highlight">
              <span className="is-bold">Giving agency to users.</span>
              Choice over what appears on your feed. Multiple feeds. A plethora of ways to filter
              content. In short, you have complete control over what you see on Seddit. (Please
              note that Seddit is a work in progress and that many of these features are yet to be
              built.)
            </div>
            <div className="about-highlight">
              <span className="is-bold">No dark patterns.</span>
              On Seddit, there are no nagging popups asking you to sign up. You don't need an
              account to simply view a page. Images, in their highest quality, can be freely
              downloaded. We don't manipulate you into using our platform more than you desire to.
            </div>
          </div>
          <div className="about-section about-faq">
            <div className="about-faq-title">Frequently asked questions</div>
            <div className="about-faq-list">{renderFaqItems()}</div>
          </div>
        </div>
      </div>
    </StaticPage>
  );
};

export default About;
