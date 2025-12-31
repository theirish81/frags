# Welcome to Frags
**Note:** The project is still in development, but you can already try it out.

## What is Frags?
Frags is an advanced AI/LLM Agent dedicated to executing complex workflows of data retrieval, transformation, extraction
and aggregation. It is designed to be highly customizable and extensible, allowing you to integrate it with your own
tools and processes. Its main goal is **precision and focus**, and it's a **system** dedicated to engineers and
specialists rather than a *code-free quick fix*.

Frags comes as a **CLI tool** and as a **library to be integrated into Golang projects**.

### Main features
* **Multi LLM:** Frags supports multiple LLMs, allowing you to choose the one that best suits your needs.
* **Dedicated almost exclusively to producing structured content:** the purpose of Frags is to be integrated in advanced
  workflows, therefore its output needs to be perfectly predictable and consumable by a machine.
* **Orchestration system:** Frags **is not** an agent to which you ask a question a simple answer back. The  whole
  purpose of Frags is to allow the user to describe complex data retrieval, transformation, extraction and  aggregation
  to produce complex data structures.
* **Advanced support for tools:** Frags has a whole standardized system to integrate with internal (as in: provided by
  the integrator) and external tools (as in: MCP servers).
* **Anti-context-bloating:** Frags multi-session system allows you to define and organize what is present in the LLM
  context, based on the session task, improving focus and reducing the risk of hallucinations
* **Output segmentation:** Frags allows you to segment your output into multiple parts, allowing you to overcome
  output token limitations, and improving answer quality.
* **Advanced pre/post-processing:** Frags allows you to define custom pre/post-processing steps, scripts, tools, and
  transformers, reducing the amount of LLM work where not necessary, reducing cost, improving performance and answer
  quality.
* **Modularity:** Frags is designed to be easily extensible, allowing you to add new features and integrate them with
  your own tools.

### Use cases
* **Research/Paper:** when your needs go beyond getting a straight answer, but need a whole structured research
  on sophisticated topics. The context reduction, focus enhancement, document ingestion combined with internet search
  allows you to design how your paper should contain in each section, allowing the LLM to focus on each  objective and
  produce data you can process however you want.
* **Data extraction:** Frags allows you to define complex data extraction pipelines, allowing you to extract data from
  documents, making sure the output is structured and predictable. This makes it easy to plug into other systems that
  expect predefined fields and values.
* **Data transformation/analysis:** From data retrieval (via the Internet, databases or any MCP tool available) to
  analysis or transformation, Frags can guide the process and provide solid data structures, skimming the context and
  increasing the credibility of the results.
* **Reporting:** Connect Frags to data sources and define complex reporting templates that describe the entire status
  of a system, division or company. Connect the output to a reporting tool to produce quality reports.
* **Notes augmentation:** give Frags your notes and design how you want the LLM to expand them into a fully featured
  document.
* **Chatbot augmentation:** improve your chatbot from an "answers machine" to an "explanation engine."
* **Creative writing:** Frags can be used to generate creative content, allowing you to design how your writing should
  look like, and how it should be structured.

## Find the full documentation in the [Frags Wiki](https://github.com/theirish81/frags/wiki)